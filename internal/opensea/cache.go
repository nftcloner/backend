package opensea

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
)

func RefreshCache(contract common.Address, tokenId uint64) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.opensea.io/api/v1/asset/%s/%d/validate/?force_update=true", contract.Hex(), tokenId), nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/97.0.4692.71 Safari/537.36")
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", res.Status, string(body))
	}

	return nil
}
