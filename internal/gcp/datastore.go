package gcp

import (
	"context"

	"cloud.google.com/go/datastore"
)

type datastoreClient struct {
	client *datastore.Client
}

func NewDatastoreClient(ctx context.Context, projectID string) (*datastoreClient, error) {
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &datastoreClient{client: client}, nil
}

func (ds *datastoreClient) Store(ctx context.Context, kind string, keyName string, data interface{}) error {
	key := datastore.NameKey(kind, keyName, nil)
	if _, err := ds.client.Put(ctx, key, data); err != nil {
		return err
	}

	return nil
}
