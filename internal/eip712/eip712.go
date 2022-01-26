package eip712

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/pkg/errors"
)

func hashTypedData(primaryType string, domain apitypes.TypedDataDomain, types apitypes.Types, message apitypes.TypedDataMessage) ([]byte, error) {
	types["EIP712Domain"] = []apitypes.Type{
		{Name: "name", Type: "string"},
		{Name: "version", Type: "string"},
		{Name: "chainId", Type: "uint256"},
		{Name: "verifyingContract", Type: "string"},
	}

	signerData := apitypes.TypedData{
		Types:       types,
		PrimaryType: primaryType,
		Domain:      domain,
		Message:     message,
	}

	typedDataHash, err := signerData.HashStruct(signerData.PrimaryType, signerData.Message)
	if err != nil {
		return nil, errors.Wrap(err, "hash typed data")
	}

	domainSeparator, err := signerData.HashStruct("EIP712Domain", signerData.Domain.Map())
	if err != nil {
		return nil, errors.Wrap(err, "hash domain separator")
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	challengeHash := crypto.Keccak256Hash(rawData)

	return challengeHash.Bytes(), nil
}

func SignTypedData(privateKey *ecdsa.PrivateKey, primaryType string, domain apitypes.TypedDataDomain, types apitypes.Types, message apitypes.TypedDataMessage) ([]byte, error) {
	challengeHash, err := hashTypedData(primaryType, domain, types, message)
	if err != nil {
		return nil, err
	}

	signature, err := crypto.Sign(challengeHash, privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "sign typed data")
	}

	if signature[64] < 2 {
		signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	}

	return signature, nil
}

func VerifyTypedData(primaryType string, domain apitypes.TypedDataDomain, types apitypes.Types, message apitypes.TypedDataMessage, signature []byte) (common.Address, error) {
	if len(signature) != 65 {
		return common.Address{}, errors.Errorf("invalid signature length: %d", len(signature))
	}

	if signature[64] == 27 || signature[64] == 28 {
		signature[64] -= 27 // Transform V from 27/28 to 0/1
	}

	challengeHash, err := hashTypedData(primaryType, domain, types, message)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "hash typed data")
	}

	pubKeyRaw, err := crypto.Ecrecover(challengeHash, signature)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "invalid signature")
	}

	pubKey, err := crypto.UnmarshalPubkey(pubKeyRaw)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "unmarshal pubkey")
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	return recoveredAddr, nil
}
