package accounts

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"

	"github.com/ebfe/scard"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var YubiKeySlots = make(map[common.Address]byte)

func connectCard() (*scard.Context, *scard.Card, error) {
	ctx, err := scard.EstablishContext()
	if err != nil {
		return nil, nil, fmt.Errorf("error establishing PC/SC connection: %v", err)
	}

	readers, err := ctx.ListReaders()
	if err != nil || len(readers) == 0 {
		ctx.Release()
		return nil, nil, fmt.Errorf("couldn't find any smart card readers: %v", err)
	}

	card, err := ctx.Connect(readers[0], scard.ShareShared, scard.ProtocolAny)
	if err != nil {
		ctx.Release()
		return nil, nil, fmt.Errorf("error connecting to card: %v", err)
	}

	return ctx, card, nil
}

func transmitAndCheck(card *scard.Card, apdu []byte, stepName string) error {
	rsp, err := card.Transmit(apdu)
	if err != nil {
		return fmt.Errorf("error in %s: %v", stepName, err)
	}
	if !isSuccess(rsp) {
		return fmt.Errorf("error on %s. Code State: %X", stepName, rsp[len(rsp)-2:])
	}
	return nil
}

func selectOpenPGP(card *scard.Card) error {
	// 0x00: CLA (Interindustry)
	// 0xA4: INS (SELECT)
	// 0x04: P1 (Select by DF Name)
	// 0x00: P2 (First or only occurrence)
	// 0x06: Lc (Length of the AID)
	// 0xD2, 0x76, 0x00, 0x01, 0x24, 0x01: Data (OpenPGP AID)
	selectApdu := []byte{0x00, 0xA4, 0x04, 0x00, 0x06, 0xD2, 0x76, 0x00, 0x01, 0x24, 0x01}
	return transmitAndCheck(card, selectApdu, "OpenPGP Applet Selection")
}

func isSuccess(rsp []byte) bool {
	return len(rsp) >= 2 && rsp[len(rsp)-2] == 0x90 && rsp[len(rsp)-1] == 0x00
}

func readPublicKey(card *scard.Card, slot byte) (common.Address, error) {
	// 0x00: CLA (Interindustry)
	// 0x47: INS (GENERATE ASYMMETRIC KEY PAIR / READ KEY)
	// 0x81: P1 (Read mode)
	// 0x00: P2 (Reserved)
	// 0x02: Lc (Length of Data)
	// slot: Data byte 1 (0xB6 o 0xA4)
	// 0x00: Data byte 2 (Reserved)
	readKeyApdu := []byte{0x00, 0x47, 0x81, 0x00, 0x02, slot, 0x00}
	rsp, err := card.Transmit(readKeyApdu)
	if err != nil || !isSuccess(rsp) {
		return common.Address{}, fmt.Errorf("slot %X empty or inaccessible", slot)
	}

	pubKeyBytes := extractPubKeyBytes(rsp[:len(rsp)-2])
	if pubKeyBytes == nil {
		return common.Address{}, fmt.Errorf("couldn't extract secp256k1 key from slot %X", slot)
	}

	pub, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return common.Address{}, err
	}

	return crypto.PubkeyToAddress(*pub), nil
}

func InitYubiKey() error {
	ctx, card, err := connectCard()
	if err != nil {
		return err
	}
	defer ctx.Release()
	defer card.Disconnect(scard.LeaveCard)

	if err := selectOpenPGP(card); err != nil {
		return err
	}

	YubiKeySlots = make(map[common.Address]byte)

	slotsToRead := []byte{0xB6, 0xA4}
	for _, slot := range slotsToRead {
		addr, err := readPublicKey(card, slot)
		if err == nil {
			YubiKeySlots[addr] = slot
		}
	}

	if len(YubiKeySlots) == 0 {
		return fmt.Errorf("couldn't find secp256k1 keys in the YubiKey.")
	}

	return nil
}

func GetYubiKeyAddresses() []common.Address {
	var addresses []common.Address
	for addr := range YubiKeySlots {
		addresses = append(addresses, addr)
	}
	return addresses
}

func normalizeSignature(sigBytes []byte) []byte {
	if len(sigBytes) != 64 {
		return sigBytes
	}

	s := new(big.Int).SetBytes(sigBytes[32:])
	N := crypto.S256().Params().N
	halfN := new(big.Int).Div(N, big.NewInt(2))

	if s.Cmp(halfN) > 0 {
		s.Sub(N, s)
		sBytes := s.Bytes()

		paddedS := make([]byte, 32)
		copy(paddedS[32-len(sBytes):], sBytes)

		copy(sigBytes[32:], paddedS)
	}

	return sigBytes
}

func SignYubiKeyTransaction(from common.Address, tx *types.Transaction, chainID *big.Int, pin string) (*types.Transaction, error) {
	slot, exists := YubiKeySlots[from]
	if !exists {
		return nil, fmt.Errorf("the address %s does not belong to this YubiKey", from.Hex())
	}

	ctx, card, err := connectCard()
	if err != nil {
		return nil, fmt.Errorf("hardware not available: %v", err)
	}
	defer ctx.Release()
	defer card.Disconnect(scard.LeaveCard)

	selectOpenPGP(card)

	// --- FIX: Dynamic PIN Mode Selection ---
	var pinP2 byte
	switch slot {
	case 0xB6:
		pinP2 = 0x81 // PW1 Mode Signature
	case 0xA4:
		pinP2 = 0x82 // PW1 Mode Global (Authentication)
	default:
		return nil, fmt.Errorf("slot not supported for PIN: %X", slot)
	}

	userPin := []byte(pin)
	// 0x00: CLA (Interindustry)
	// 0x20: INS (VERIFY)
	// 0x00: P1 (Reserved)
	// pinP2: P2 (0x81 PW1 Signature o 0x82 PW1 Authentication)
	// byte(len(userPin)): Lc (Length of the PIN)
	verifyPinApdu := append([]byte{0x00, 0x20, 0x00, pinP2, byte(len(userPin))}, userPin...)
	err = transmitAndCheck(card, verifyPinApdu, fmt.Sprintf("User PIN verification (Mode %X)", pinP2))
	if err != nil {
		return nil, fmt.Errorf("error verifying PIN: %v", err)
	}
	// ---------------------------------------

	signer := types.LatestSignerForChainID(chainID)
	txHash := signer.Hash(tx)

	var signApdu []byte
	var operationName string

	switch slot {
	case 0xB6:
		// 0x00: CLA (Interindustry)
		// 0x2A: INS (PERFORM SECURITY OPERATION)
		// 0x9E: P1 (Compute Digital Signature)
		// 0x9A: P2 (Input data tag)
		// byte(len(txHash.Bytes())): Lc (Length of transaction hash)
		signApdu = append([]byte{0x00, 0x2A, 0x9E, 0x9A, byte(len(txHash.Bytes()))}, txHash.Bytes()...)
		operationName = "Signature Slot Sign"
	case 0xA4:
		// 0x00: CLA (Interindustry)
		// 0x88: INS (INTERNAL AUTHENTICATE)
		// 0x00: P1 (Reserved)
		// 0x00: P2 (Reserved)
		// byte(len(txHash.Bytes())): Lc (Length of transaction hash)
		signApdu = append([]byte{0x00, 0x88, 0x00, 0x00, byte(len(txHash.Bytes()))}, txHash.Bytes()...)
		operationName = "Authentication Slot Sign"
	default:
		return nil, fmt.Errorf("slot not supported for signing: %X", slot)
	}

	fmt.Printf("Please touch your YubiKey to sign the transaction (%s)...\n", operationName)

	sigRsp, err := card.Transmit(signApdu)
	if err != nil {
		return nil, fmt.Errorf("error transmitting APDU: %v", err)
	}

	// Handle long buffers (GET RESPONSE for state 61XX)
	if len(sigRsp) >= 2 && sigRsp[len(sigRsp)-2] == 0x61 {
		bytesWaiting := sigRsp[len(sigRsp)-1]
		// 0x00: CLA (Interindustry)
		// 0xC0: INS (GET RESPONSE)
		// 0x00: P1 (Reserved)
		// 0x00: P2 (Reserved)
		// bytesWaiting: Le (Expected length)
		getResponseApdu := []byte{0x00, 0xC0, 0x00, 0x00, bytesWaiting}

		rspData := sigRsp[:len(sigRsp)-2]

		newRsp, err := card.Transmit(getResponseApdu)
		if err != nil {
			return nil, fmt.Errorf("error executing GET RESPONSE: %v", err)
		}

		sigRsp = append(rspData, newRsp...)
	}

	if !isSuccess(sigRsp) {
		return nil, fmt.Errorf("the card rejected the operation. State: %X", sigRsp[len(sigRsp)-2:])
	}

	signature := normalizeSignature(sigRsp[:len(sigRsp)-2])

	var finalSignature []byte
	for v := 0; v < 2; v++ {
		sigWithV := append(signature, byte(v))
		recoveredPub, err := crypto.SigToPub(txHash.Bytes(), sigWithV)
		if err == nil {
			if crypto.PubkeyToAddress(*recoveredPub) == from {
				finalSignature = sigWithV
				break
			}
		}
	}

	if finalSignature == nil {
		return nil, fmt.Errorf("couldn't derive V parameter. Raw signature might need to be parsed from ASN.1 DER.")
	}

	return tx.WithSignature(signer, finalSignature)
}

func extractPubKeyBytes(tlv []byte) []byte {
	for i := 0; i < len(tlv)-2; i++ {
		if tlv[i] == 0x86 {
			length := int(tlv[i+1])
			if i+2+length <= len(tlv) && tlv[i+2] == 0x04 {
				return tlv[i+2 : i+2+length]
			}
		}
	}
	return nil
}

func GenerateKey() {
	ctx, card, err := connectCard()
	if err != nil {
		log.Fatalf("hardware not available: %v", err)
	}
	defer ctx.Release()
	defer card.Disconnect(scard.LeaveCard)

	selectOpenPGP(card)

	slots := []byte{0xB6, 0xA4}
	var targetSlot byte = 0x00

	for _, slot := range slots {
		_, err := readPublicKey(card, slot)
		if err != nil {
			targetSlot = slot
			break
		}
	}

	if targetSlot == 0x00 {
		fmt.Println("\n[!] OPERATION ABORTED: No available slots found.")
		fmt.Println("Every slot is already in use.")
		fmt.Println("To generate a new account, you must manually reset the card using the Factory Reset command.")
		return
	}

	fmt.Printf("\nAvailable slot found: %X. Starting generation...\n", targetSlot)

	adminPin := []byte("12345678")
	// 0x00: CLA (Interindustry)
	// 0x20: INS (VERIFY)
	// 0x00: P1 (Reserved)
	// 0x83: P2 (Admin PIN PW3)
	// byte(len(adminPin)): Lc (Length of the PIN)
	verifyPinApdu := append([]byte{0x00, 0x20, 0x00, 0x83, byte(len(adminPin))}, adminPin...)
	transmitAndCheck(card, verifyPinApdu, "Admin PIN verification")

	var algoTag byte
	switch targetSlot {
	case 0xB6:
		algoTag = 0xC1
	case 0xA4:
		algoTag = 0xC3
	default:
		log.Fatalf("Error: Slot not supported (%X)\n", targetSlot)
	}

	// 0x00: CLA (Interindustry)
	// 0xDA: INS (PUT DATA)
	// 0x00: P1 (Reserved)
	// algoTag: P2 (Algorithm Tag C1 or C3)
	// 0x06: Lc (Length of algorithm data)
	// 0x13, 0x2B, 0x81, 0x04, 0x00, 0x0A: Data (secp256k1 OID)
	setAlgoApdu := []byte{0x00, 0xDA, 0x00, algoTag, 0x06, 0x13, 0x2B, 0x81, 0x04, 0x00, 0x0A}
	transmitAndCheck(card, setAlgoApdu, fmt.Sprintf("secp256k1 algorithm selection for slot %X", targetSlot))

	// 0x00: CLA (Interindustry)
	// 0x47: INS (GENERATE ASYMMETRIC KEY PAIR)
	// 0x80: P1 (Generate mode)
	// 0x00: P2 (Reserved)
	// 0x02: Lc (Length of Data)
	// targetSlot: Data byte 1 (0xB6 o 0xA4)
	// 0x00: Data byte 2 (Reserved)
	genKeyApdu := []byte{0x00, 0x47, 0x80, 0x00, 0x02, targetSlot, 0x00}
	rsp, err := card.Transmit(genKeyApdu)
	if err != nil {
		log.Fatalf("Error transmitting APDU: %v\n", err)
	}

	if len(rsp) >= 2 && rsp[len(rsp)-2] == 0x61 {
		bytesWaiting := rsp[len(rsp)-1]
		// 0x00: CLA (Interindustry)
		// 0xC0: INS (GET RESPONSE)
		// 0x00: P1 (Reserved)
		// 0x00: P2 (Reserved)
		// bytesWaiting: Le (Expected length)
		getResponseApdu := []byte{0x00, 0xC0, 0x00, 0x00, bytesWaiting}

		rspData := rsp[:len(rsp)-2]

		newRsp, err := card.Transmit(getResponseApdu)
		if err != nil {
			fmt.Printf("Error executing GET RESPONSE: %v\n", err)
			return
		}

		rsp = append(rspData, newRsp...)
	}

	if isSuccess(rsp) {
		fmt.Println("Key generated successfully in the YubiKey.")

		pubBytes := extractPubKeyBytes(rsp[:len(rsp)-2])
		if pubBytes != nil {
			fmt.Printf("Complete Public Key (Hex): %s\n", hex.EncodeToString(pubBytes))
		} else {
			fmt.Println("Error: Could not extract public key from response.")
		}
	} else {
		log.Fatalf("Error generating key. State: %X\n", rsp[len(rsp)-2:])
	}
}
