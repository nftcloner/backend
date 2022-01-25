package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/storage"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/nftcloner/backend/internal/eip712"
)

func UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		response(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}

	contract := r.Form.Get("contract")
	tokenId := r.Form.Get("tokenId")
	signature := r.Form.Get("signature")

	if contract == "" {
		response(w, http.StatusBadRequest, M{"error": "contract is required"})
		return
	}

	if tokenId == "" {
		response(w, http.StatusBadRequest, M{"error": "tokenId is required"})
		return
	}

	if signature == "" {
		response(w, http.StatusBadRequest, M{"error": "signature is required"})
		return
	}

	tokenIdInt, err := strconv.ParseInt(tokenId, 10, 64)
	if err != nil {
		response(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}

	signatureBytes, err := hexutil.Decode(signature)
	if err != nil {
		response(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}

	userEOA, err := VerifySignature(contract, tokenIdInt, signatureBytes)
	if err != nil {
		response(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}

	userTokenId, err := GetTokenIdFromContract(r.Context(), userEOA)
	if err != nil {
		response(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}
	userTokenIdStr := strconv.FormatInt(userTokenId, 10)

	if contract == "" || tokenId == "" {
		response(w, http.StatusBadRequest, M{"error": "contract and tokenId are required"})
		return
	}

	metadataInfo, err := GetMetadataURIFromOpenSea(contract, tokenId)
	if err != nil {
		response(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}

	metadata, err := DownloadMetadata(metadataInfo.TokenMetadata)
	if err != nil {
		response(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}

	if err = StoreInGCPDatastore(r.Context(), userTokenIdStr, userEOA, contract, tokenId, metadataInfo.Name, metadataInfo.TokenMetadata); err != nil {
		response(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}

	if err = StoreInGCPStorage(r.Context(), userTokenIdStr, metadata); err != nil {
		response(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}

	response(w, http.StatusOK, string(metadata))
}

func VerifySignature(contract string, tokenId int64, signature []byte) (common.Address, error) {
	primaryType := "UpdateMetadata"
	domain := apitypes.TypedDataDomain{
		Name:              "NFTCloner",
		Version:           "1",
		ChainId:           math.NewHexOrDecimal256(1),
		VerifyingContract: os.Getenv("NFT_CONTRACT_ADDRESS"),
	}
	types := apitypes.Types{
		"UpdateMetadata": []apitypes.Type{
			{Name: "contract", Type: "address"},
			{Name: "tokenId", Type: "uint256"},
		},
	}
	message := apitypes.TypedDataMessage{
		"contract": common.HexToAddress(contract).Hex(),
		"tokenId":  math.NewHexOrDecimal256(tokenId),
	}

	signer, err := eip712.VerifyTypedData(
		primaryType,
		domain,
		types,
		message,
		signature,
	)
	if err != nil {
		return common.Address{}, err
	}

	return signer, nil
}

func GetTokenIdFromContract(ctx context.Context, owner common.Address) (int64, error) {
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

type OpenSeaAsset struct {
	Name          string `json:"name"`
	TokenMetadata string `json:"token_metadata"`
}

func GetMetadataURIFromOpenSea(contract string, tokenId string) (*OpenSeaAsset, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.opensea.io/api/v1/asset/%s/%s/", contract, tokenId), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/97.0.4692.71 Safari/537.36")
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", res.Status, string(body))
	}

	var openSeaAsset OpenSeaAsset
	if err = json.Unmarshal(body, &openSeaAsset); err != nil {
		return nil, err
	}

	return &openSeaAsset, nil
}

func DownloadMetadata(uri string) ([]byte, error) {
	if strings.HasPrefix(uri, "ipfs://") {
		// TODO: handle IPFS
		return nil, errors.New("ipfs not supported")
	}

	res, err := http.Get(uri)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", res.Status, string(body))
	}

	return body, nil
}

type TokenMetadata struct {
	EOA         string    `datastore:"eoa"`
	Name        string    `datastore:"name"`
	Contract    string    `datastore:"contract"`
	TokenId     string    `datastore:"token_id"`
	MetadataURI string    `datastore:"metadata_uri"`
	Time        time.Time `datastore:"time"`
}

func StoreInGCPDatastore(ctx context.Context, userTokenId string, userEOA common.Address, contract string, tokenId string, tokenName string, metadataURI string) error {
	client, err := datastore.NewClient(ctx, "nftcloner")
	if err != nil {
		return err
	}

	key := datastore.NameKey("TokenMetadata", userTokenId, nil)
	if _, err = client.Put(ctx, key, &TokenMetadata{
		EOA:         userEOA.Hex(),
		Name:        tokenName,
		Contract:    contract,
		TokenId:     tokenId,
		MetadataURI: metadataURI,
		Time:        time.Now(),
	}); err != nil {
		return err
	}

	return nil
}

func StoreInGCPStorage(ctx context.Context, tokenId string, metadata []byte) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	obj := client.Bucket("cdn.nftcloner.xyz").Object(fmt.Sprintf("nft/%s.json", tokenId))
	w := obj.NewWriter(ctx)
	if _, err = w.Write(metadata); err != nil {
		return err
	}

	if err = w.Close(); err != nil {
		return err
	}

	if err = obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return err
	}

	return nil
}
