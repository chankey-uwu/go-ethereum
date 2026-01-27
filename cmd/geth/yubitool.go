package main

import (
	"fmt"

	"github.com/go-piv/piv-go/piv"
	"github.com/urfave/cli/v2"
)

var yubiToolCommand = &cli.Command{
	Name:  "yubitool",
	Usage: "Manage YubiKey: List and Generate Keys",
	Action: func(c *cli.Context) error {

		cards, err := piv.Cards()
		if err != nil {
			return fmt.Errorf("Error listing cards: %w", err)
		}
		if len(cards) == 0 {
			fmt.Println("No YubiKeys found.")
			return nil
		}

		fmt.Printf("Found YubiKey in reader: %s\n", cards[0])

		yk, err := piv.Open(cards[0])
		if err != nil {
			return fmt.Errorf("error connecting card: %w", err)
		}
		defer yk.Close()

		key := piv.Key{
			Algorithm:   piv.AlgorithmEC256,
			PINPolicy:   piv.PINPolicyNever,
			TouchPolicy: piv.TouchPolicyNever,
		}

		pubKey, err := yk.GenerateKey(piv.DefaultManagementKey, piv.SlotAuthentication, key)
		if err != nil {
			return fmt.Errorf("error generating key: %w", err)
		}

		fmt.Printf("Public Key: %+v\n", pubKey)

		return nil
	},
}
