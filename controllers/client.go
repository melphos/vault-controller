package controllers

import (
	"github.com/hashicorp/vault/api"
	vaultapi "github.com/hashicorp/vault/api"
)

func GetClient(address, token string) (*vaultapi.Client, error) {
	config := api.Config{}

	config.ConfigureTLS(&api.TLSConfig{
		Insecure: true,
	})

	vclient, err := vaultapi.NewClient(&config)
	if err != nil {
		return nil, err
	}
	vclient.SetAddress(address)
	vclient.SetToken(token)
	return vclient, nil
}
