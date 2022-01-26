package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/nftcloner/backend/internal/eip712"
	"github.com/nftcloner/backend/internal/gcp"
	"github.com/nftcloner/backend/internal/metadata"
	"github.com/nftcloner/backend/internal/nftcloner"
	"github.com/nftcloner/backend/internal/opensea"
)

type MetadataInfo struct {
	EOA         string    `datastore:"eoa"`
	Name        string    `datastore:"name"`
	Contract    string    `datastore:"contract"`
	TokenId     string    `datastore:"token_id"`
	MetadataURI string    `datastore:"metadata_uri"`
	Time        time.Time `datastore:"time"`
}

func UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := r.ParseForm(); err != nil {
		response(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}

	ctxValues := logrus.Fields{}

	contractForm := r.Form.Get("contract")
	tokenIdForm := r.Form.Get("tokenId")
	signatureForm := r.Form.Get("signature")

	ctxValues["contract"] = contractForm
	ctxValues["tokenId"] = tokenIdForm
	ctxValues["signature"] = signatureForm

	if contractForm == "" {
		errorResponse(w, http.StatusBadRequest, errors.New("contract is required"), ctxValues)
		return
	}
	contract := common.HexToAddress(contractForm)
	ctxValues["contract"] = contract.Hex()

	if tokenIdForm == "" {
		errorResponse(w, http.StatusBadRequest, errors.New("tokenId is required"), ctxValues)
		return
	}
	tokenId, err := strconv.ParseUint(tokenIdForm, 10, 64)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err)
		return
	}
	ctxValues["tokenId"] = tokenId

	if signatureForm == "" {
		errorResponse(w, http.StatusBadRequest, errors.New("signature is required"), ctxValues)
		return
	}
	signature, err := hexutil.Decode(signatureForm)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, errors.Wrap(err, "signature is not hex"), ctxValues)
		return
	}

	userEOA, err := VerifySignature(contract, tokenId, signature)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, errors.Wrap(err, "failed to verify signature"), ctxValues)
		return
	}
	ctxValues["eoa"] = userEOA.Hex()

	userTokenId, err := nftcloner.TokenByOwner(r.Context(), userEOA)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "failed to get tokenId"), ctxValues)
		return
	}
	userTokenIdStr := strconv.FormatInt(userTokenId, 10)
	ctxValues["userTokenId"] = userTokenId

	datastoreClient, err := gcp.NewDatastoreClient(r.Context(), "nftcloner")
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "failed to create datastore client"), ctxValues)
		return
	}

	storageClient, err := gcp.NewStorageClient(r.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "failed to create storage client"), ctxValues)
		return
	}

	metadataInfo, err := opensea.GetMetadataInfo(contract, tokenId)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "failed to get metadata info"), ctxValues)
		return
	}
	ctxValues["metadataInfo"] = metadataInfo

	metadataURL, err := url.Parse(metadataInfo.TokenMetadata)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "failed to parse metadata url"), ctxValues)
		return
	}

	var rawMetadata []byte
	switch metadataURL.Scheme {
	case "http", "https":
		rawMetadata, err = metadata.HTTPDownload(metadataInfo.TokenMetadata)
	case "ipfs":
		rawMetadata, err = metadata.IPFSDownload(metadataInfo.TokenMetadata)
	default:
		errorResponse(w, http.StatusInternalServerError, errors.New("unsupported scheme"), ctxValues)
		return
	}
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "failed to download metadata"), ctxValues)
		return
	}

	if err = datastoreClient.Store(r.Context(), "metadata", userTokenIdStr, &MetadataInfo{
		EOA:         userEOA.Hex(),
		Name:        metadataInfo.Name,
		Contract:    contract.Hex(),
		TokenId:     userTokenIdStr,
		MetadataURI: metadataInfo.TokenMetadata,
		Time:        time.Now(),
	}); err != nil {
		errorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "failed to store metadata info"), ctxValues)
		return
	}

	if err = storageClient.Store(r.Context(), "cdn.nftcloner.xyz", fmt.Sprintf("nft/%d.json", tokenId), rawMetadata, true); err != nil {
		errorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "failed to store metadata"), ctxValues)
		return
	}

	// TODO: refresh cache GCP Storage & CloudFlare

	if err = opensea.RefreshCache(contract, tokenId); err != nil {
		errorResponse(w, http.StatusInternalServerError, errors.Wrap(err, "failed to refresh cache"), ctxValues)
		return
	}

	response(w, http.StatusOK, string(rawMetadata))
}

func VerifySignature(contract common.Address, tokenId uint64, signature []byte) (common.Address, error) {
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
		"contract": contract.Hex(),
		"tokenId":  math.NewHexOrDecimal256(int64(tokenId)),
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
