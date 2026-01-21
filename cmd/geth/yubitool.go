package main

import (
	"fmt"

	"github.com/go-piv/piv-go/piv"
	"github.com/urfave/cli/v2"
)

var yubiToolCommand = &cli.Command{
	Name:  "yubitool",
	Usage: "Gestionar YubiKey: Listar y Generar Keys",
	Action: func(c *cli.Context) error {

		cards, err := piv.Cards()
		if err != nil {
			return fmt.Errorf("error listando tarjetas: %w", err)
		}
		if len(cards) == 0 {
			fmt.Println("No se encontraron YubiKeys conectadas.")
			return nil
		}

		fmt.Printf("Encontrada YubiKey en lector: %s\n", cards[0])

		yk, err := piv.Open(cards[0])
		if err != nil {
			return fmt.Errorf("error conectando a la tarjeta: %w", err)
		}
		defer yk.Close()

		fmt.Println("Intentando generar llave ECC P256 en Slot 9A (Authentication)...")

		key := piv.Key{
			Algorithm:   piv.AlgorithmEC256,
			PINPolicy:   piv.PINPolicyNever,
			TouchPolicy: piv.TouchPolicyNever,
		}

		pubKey, err := yk.GenerateKey(piv.DefaultManagementKey, piv.SlotAuthentication, key)
		if err != nil {
			return fmt.Errorf("fallo al generar la key: %w", err)
		}

		fmt.Printf("¡Éxito! Llave generada.\nPublic Key: %+v\n", pubKey)

		return nil
	},
}
