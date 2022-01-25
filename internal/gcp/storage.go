package gcp

import (
	"context"

	"cloud.google.com/go/storage"
)

type storageClient struct {
	client *storage.Client
}

func NewStorageClient(ctx context.Context) (*storageClient, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &storageClient{
		client: client,
	}, nil
}

func (sc *storageClient) Store(ctx context.Context, bucket, object string, data []byte, public bool) error {
	obj := sc.client.Bucket(bucket).Object(object)

	wc := obj.NewWriter(ctx)
	_, err := wc.Write(data)
	if err != nil {
		return err
	}

	if err = wc.Close(); err != nil {
		return err
	}

	if public {
		if err = obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
			return err
		}
	}

	return nil
}
