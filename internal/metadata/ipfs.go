package metadata

import (
	"github.com/pkg/errors"
)

func IPFSDownload(url string) ([]byte, error) {
	return nil, errors.New("ipfs not implemented")
	//res, err := http.Get(url)
	//if err != nil {
	//	return nil, err
	//}
	//
	//defer res.Body.Close()
	//
	//body, err := ioutil.ReadAll(res.Body)
	//if err != nil {
	//	return nil, err
	//}
	//
	//if res.StatusCode != http.StatusOK {
	//	return nil, fmt.Errorf("%s: %s", res.Status, string(body))
	//}
	//
	//return body, nil
}
