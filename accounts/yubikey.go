package accounts

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"

	"github.com/ebfe/scard"
	"github.com/ethereum/go-ethereum/crypto"
)

func connectCard() (*scard.Context, *scard.Card) {
	ctx, err := scard.EstablishContext()
	if err != nil {
		log.Fatalf("Error al establecer contexto PC/SC: %v\n", err)
	}

	readers, err := ctx.ListReaders()
	if err != nil || len(readers) == 0 {
		log.Fatalf("No se encontraron lectores conectados.\n")
	}

	card, err := ctx.Connect(readers[0], scard.ShareShared, scard.ProtocolAny)
	if err != nil {
		log.Fatalf("Error al conectar con la tarjeta: %v\n", err)
	}
	return ctx, card
}

// selectOpenPGP envia el APDU para entrar al applet OpenPGP
func selectOpenPGP(card *scard.Card) {
	selectApdu := []byte{0x00, 0xA4, 0x04, 0x00, 0x06, 0xD2, 0x76, 0x00, 0x01, 0x24, 0x01}
	transmitAndCheck(card, selectApdu, "Seleccion Applet OpenPGP")
}

// transmitAndCheck envia un comando y aborta si la respuesta no es 90 00 (Exito)
func transmitAndCheck(card *scard.Card, apdu []byte, stepName string) {
	rsp, err := card.Transmit(apdu)
	if err != nil {
		log.Fatalf("Error en %s: %v\n", stepName, err)
	}
	if !isSuccess(rsp) {
		log.Fatalf("Fallo en %s. Codigo de estado: %X\n", stepName, rsp[len(rsp)-2:])
	}
	fmt.Printf("%s completado exitosamente.\n", stepName)
}

// isSuccess verifica los ultimos dos bytes de la respuesta
func isSuccess(rsp []byte) bool {
	return len(rsp) >= 2 && rsp[len(rsp)-2] == 0x90 && rsp[len(rsp)-1] == 0x00
}

func verifySignature(mensaje string, pubKeyHex string, sigHex string) bool {
	// 1. Decodificar la llave publica de hexadecimal a bytes
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		fmt.Printf("Error al decodificar la llave publica: %v\n", err)
		return false
	}

	// 2. Decodificar la firma en crudo (64 bytes: R y S concatenados)
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		fmt.Printf("Error al decodificar la firma: %v\n", err)
		return false
	}

	// 3. Validar que tengamos los 64 bytes exactos
	if len(sigBytes) != 64 {
		fmt.Printf("Error: Se esperaban 64 bytes de firma, se recibieron %d\n", len(sigBytes))
		return false
	}

	// 4. Recrear el hash Keccak-256 del mensaje original
	hash := crypto.Keccak256Hash([]byte(mensaje))

	// 5. Verificar matematicamente usando go-ethereum
	isValid := crypto.VerifySignature(pubKeyBytes, hash.Bytes(), sigBytes)

	return isValid
}

func generateKey() {
	// Conectar a la YubiKey
	ctx, card := connectCard()
	defer ctx.Release()
	defer card.Disconnect(scard.LeaveCard)

	selectOpenPGP(card)

	// 1. Verificar Admin PIN (PIN3)
	adminPin := []byte("12345678") // Cambiar si es distinto
	fmt.Println("Verificando Admin PIN...")
	verifyPinApdu := append([]byte{0x00, 0x20, 0x00, 0x83, byte(len(adminPin))}, adminPin...)
	transmitAndCheck(card, verifyPinApdu, "Verificacion PIN Administrador")

	// 2. Configurar atributo de algoritmo a secp256k1 en el slot C1
	fmt.Println("Configurando algoritmo secp256k1 en el slot de firma...")
	setAlgoApdu := []byte{0x00, 0xDA, 0x00, 0xC1, 0x06, 0x13, 0x2B, 0x81, 0x04, 0x00, 0x0A}
	transmitAndCheck(card, setAlgoApdu, "Cambio de Algoritmo a secp256k1")

	// 3. Generar la llave
	fmt.Println("Generando llave secp256k1 (esto tomara unos segundos)...")
	genKeyApdu := []byte{0x00, 0x47, 0x80, 0x00, 0x02, 0xB6, 0x00}
	rsp, err := card.Transmit(genKeyApdu)
	if err != nil {
		log.Fatalf("Error al transmitir APDU de generacion: %v\n", err)
	}

	if isSuccess(rsp) {
		fmt.Println("Llave generada con exito.")
		extraerLlavePublica(rsp[:len(rsp)-2])
	} else {
		log.Fatalf("Fallo la generacion de la llave. Estado: %X\n", rsp[len(rsp)-2:])
	}
}

// extraerLlavePublica parsea el TLV y formatea la salida en hexadecimal
func extraerLlavePublica(tlv []byte) {
	fmt.Println("Buscando llave publica en la respuesta...")
	for i := 0; i < len(tlv)-2; i++ {
		// Buscamos el Tag 86 que contiene la llave publica
		if tlv[i] == 0x86 {
			length := int(tlv[i+1])
			if i+2+length <= len(tlv) && tlv[i+2] == 0x04 {
				pubKeyBytes := tlv[i+2 : i+2+length]
				fmt.Printf("Llave Publica Completa (Hex): %s\n", hex.EncodeToString(pubKeyBytes))
				fmt.Printf("Coordenada X (Hex): %s\n", hex.EncodeToString(pubKeyBytes[1:33]))
				fmt.Printf("Coordenada Y (Hex): %s\n", hex.EncodeToString(pubKeyBytes[33:65]))
				return
			}
		}
	}
	fmt.Println("Error: No se encontro la llave publica en el bloque TLV.")
}

func normalizeSignature(sigBytes []byte) []byte {
	if len(sigBytes) != 64 {
		return sigBytes
	}

	// Extraemos el valor S de la firma (ultimos 32 bytes)
	s := new(big.Int).SetBytes(sigBytes[32:])

	// Obtenemos el orden de la curva secp256k1 (N) directo de go-ethereum
	N := crypto.S256().Params().N

	// Calculamos la mitad exacta de la curva (N/2)
	halfN := new(big.Int).Div(N, big.NewInt(2))

	// Si S es mayor que N/2, invertimos S restandolo del total N: S' = N - S
	if s.Cmp(halfN) > 0 {
		s.Sub(N, s)
		sBytes := s.Bytes()

		// Aseguramos que el nuevo S tenga siempre 32 bytes (padding con 0 a la izquierda si es necesario)
		paddedS := make([]byte, 32)
		copy(paddedS[32-len(sBytes):], sBytes)

		// Sobrescribimos el valor S original en los bytes de la firma
		copy(sigBytes[32:], paddedS)
		fmt.Println("INFO: Firma normalizada a 'Low S' (Estandar Ethereum EIP-2).")
	}

	return sigBytes
}

// sign toma un mensaje, genera su hash Keccak-256 y le pide a la YubiKey que lo firme.
func sign(mensaje string) string {
	ctx, card := connectCard()
	defer ctx.Release()
	defer card.Disconnect(scard.LeaveCard)

	selectOpenPGP(card)

	userPin := []byte("123456")
	fmt.Println("Verificando User PIN...")
	verifyUserPinApdu := append([]byte{0x00, 0x20, 0x00, 0x81, byte(len(userPin))}, userPin...)
	transmitAndCheck(card, verifyUserPinApdu, "Verificacion User PIN")

	fmt.Printf("Generando hash Keccak-256 para el mensaje: '%s'\n", mensaje)
	hash := crypto.Keccak256Hash([]byte(mensaje))

	fmt.Printf("Hash resultante (Hex): %s\n", hash.Hex())

	fmt.Println("Enviando hash a la YubiKey para firmar...")
	signApdu := append([]byte{0x00, 0x2A, 0x9E, 0x9A, byte(len(hash.Bytes()))}, hash.Bytes()...)

	sigRsp, err := card.Transmit(signApdu)
	if err != nil {
		log.Fatalf("Error al transmitir APDU de firma: %v\n", err)
	}

	if isSuccess(sigRsp) {
		signature := sigRsp[:len(sigRsp)-2]

		// APLICAMOS LA NORMALIZACION DE ETHEREUM AQUI
		signature = normalizeSignature(signature)

		sigHex := hex.EncodeToString(signature)

		fmt.Println("Firma generada exitosamente por el hardware.")
		fmt.Printf("Firma en bruto (Hex): %s\n", sigHex)

		return sigHex
	} else {
		log.Fatalf("Fallo la firma. Codigo de estado: %X\n", sigRsp[len(sigRsp)-2:])
		return ""
	}
}
