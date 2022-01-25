package opensea

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
)

type Asset struct {
	Name          string `json:"name"`
	TokenMetadata string `json:"token_metadata"`
}

func GetMetadataInfo(contract common.Address, tokenId uint64) (*Asset, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.opensea.io/api/v1/asset/%s/%d/", contract, tokenId), nil)
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

	var openSeaAsset Asset
	if err = json.Unmarshal(body, &openSeaAsset); err != nil {
		return nil, err
	}

	return &openSeaAsset, nil
}
