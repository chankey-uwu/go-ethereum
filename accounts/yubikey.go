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

var YubiKeyAddress common.Address

func connectCard() (*scard.Context, *scard.Card, error) {
	ctx, err := scard.EstablishContext()
	if err != nil {
		return nil, nil, fmt.Errorf("error stablishing PC/SC connection: %v", err)
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

func selectOpenPGP(card *scard.Card) error {
	selectApdu := []byte{0x00, 0xA4, 0x04, 0x00, 0x06, 0xD2, 0x76, 0x00, 0x01, 0x24, 0x01}
	return transmitAndCheck(card, selectApdu, "Seleccion Applet OpenPGP")
}

func transmitAndCheck(card *scard.Card, apdu []byte, stepName string) error {
	rsp, err := card.Transmit(apdu)
	if err != nil {
		return fmt.Errorf("error in %s: %v", stepName, err)
	}
	if !isSuccess(rsp) {
		return fmt.Errorf("error om %s. Code State: %X", stepName, rsp[len(rsp)-2:])
	}
	return nil
}

func isSuccess(rsp []byte) bool {
	return len(rsp) >= 2 && rsp[len(rsp)-2] == 0x90 && rsp[len(rsp)-1] == 0x00
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

	readKeyApdu := []byte{0x00, 0x47, 0x81, 0x00, 0x02, 0xB6, 0x00}
	rsp, err := card.Transmit(readKeyApdu)
	if err != nil || !isSuccess(rsp) {
		return fmt.Errorf("error reading public key from YubiKey: %v", err)
	}

	pubKeyBytes := extractPubKeyBytes(rsp[:len(rsp)-2])
	if pubKeyBytes == nil {
		return fmt.Errorf("couldn't extract secp256k1 key from YubiKey response")
	}

	pub, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return err
	}

	YubiKeyAddress = crypto.PubkeyToAddress(*pub)
	return nil
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

func SignYubiKeyTransaction(tx *types.Transaction, chainID *big.Int, pin string) (*types.Transaction, error) {
	ctx, card, err := connectCard()
	if err != nil {
		return nil, fmt.Errorf("hardware not available: %v", err)
	}
	defer ctx.Release()
	defer card.Disconnect(scard.LeaveCard)

	selectOpenPGP(card)

	userPin := []byte(pin)
	verifyPinApdu := append([]byte{0x00, 0x20, 0x00, 0x81, byte(len(userPin))}, userPin...)
	transmitAndCheck(card, verifyPinApdu, "User PIN verification")

	signer := types.LatestSignerForChainID(chainID)
	txHash := signer.Hash(tx)

	signApdu := append([]byte{0x00, 0x2A, 0x9E, 0x9A, byte(len(txHash.Bytes()))}, txHash.Bytes()...)
	sigRsp, err := card.Transmit(signApdu)
	if err != nil || !isSuccess(sigRsp) {
		return nil, fmt.Errorf("failed to sign transaction in YubiKey: %v", err)
	}

	signature := normalizeSignature(sigRsp[:len(sigRsp)-2])

	var finalSignature []byte
	for v := 0; v < 2; v++ {
		sigWithV := append(signature, byte(v))
		recoveredPub, err := crypto.SigToPub(txHash.Bytes(), sigWithV)
		if err == nil {
			if crypto.PubkeyToAddress(*recoveredPub) == YubiKeyAddress {
				finalSignature = sigWithV
				break
			}
		}
	}

	if finalSignature == nil {
		return nil, fmt.Errorf("couldn't derive V parameter")
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

func extractPubKey(tlv []byte) {
	for i := 0; i < len(tlv)-2; i++ {
		if tlv[i] == 0x86 {
			length := int(tlv[i+1])
			if i+2+length <= len(tlv) && tlv[i+2] == 0x04 {
				pubKeyBytes := tlv[i+2 : i+2+length]
				fmt.Printf("Complete Public Key (Hex): %s\n", hex.EncodeToString(pubKeyBytes))
				fmt.Printf("X coordinate (Hex): %s\n", hex.EncodeToString(pubKeyBytes[1:33]))
				fmt.Printf("Y coordinate (Hex): %s\n", hex.EncodeToString(pubKeyBytes[33:65]))
				return
			}
		}
	}
	fmt.Println("Error: Couldn't extract public key from TLV.")
}

func generateKey() {
	ctx, card, err := connectCard()
	if err != nil {
		log.Fatalf("hardware not available while trying to sign: %v", err)
	}
	defer ctx.Release()
	defer card.Disconnect(scard.LeaveCard)

	selectOpenPGP(card)

	adminPin := []byte("12345678")
	verifyPinApdu := append([]byte{0x00, 0x20, 0x00, 0x83, byte(len(adminPin))}, adminPin...)
	transmitAndCheck(card, verifyPinApdu, "Admin PIN verification")

	setAlgoApdu := []byte{0x00, 0xDA, 0x00, 0xC1, 0x06, 0x13, 0x2B, 0x81, 0x04, 0x00, 0x0A}
	transmitAndCheck(card, setAlgoApdu, "secp256k1 algorithm selection")

	genKeyApdu := []byte{0x00, 0x47, 0x80, 0x00, 0x02, 0xB6, 0x00}
	rsp, err := card.Transmit(genKeyApdu)
	if err != nil {
		log.Fatalf("Error transmiting APDU: %v\n", err)
	}

	if isSuccess(rsp) {
		fmt.Println("Key generated successfully.")
		extractPubKey(rsp[:len(rsp)-2])
	} else {
		log.Fatalf("Error generating key. State: %X\n", rsp[len(rsp)-2:])
	}
}
