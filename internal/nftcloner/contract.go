package nftcloner

import (
	"context"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TokenByOwner(ctx context.Context, owner common.Address) (int64, error) {
	client, err := ethclient.DialContext(ctx, os.Getenv("ETHEREUM_NODE_URL"))
	if err != nil {
		return 0, err
	}

	contractABI, err := abi.JSON(strings.NewReader(`[{"inputs":[{"internalType":"address","name":"_owner","type":"address"}],"name":"tokenByOwner","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`))
	if err != nil {
		return 0, err
	}

	callData, err := contractABI.Pack("tokenByOwner", owner)
	if err != nil {
		return 0, err
	}

	contractAddress := common.HexToAddress(os.Getenv("NFT_CONTRACT_ADDRESS"))

	result, err := client.CallContract(ctx, ethereum.CallMsg{
		From: owner,
		To:   &contractAddress,
		Data: callData,
	}, nil)
	if err != nil {
		return 0, err
	}

	outputs, err := contractABI.Unpack("tokenByOwner", result)
	if err != nil {
		return 0, err
	}

	return (outputs[0].(*big.Int)).Int64(), nil
}
