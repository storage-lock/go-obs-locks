package obs_locks

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	obs_storage "github.com/storage-lock/go-obs-storage"
	"github.com/storage-lock/go-storage"
	storage_lock "github.com/storage-lock/go-storage-lock"
	"github.com/stretchr/testify/assert"
)

// testOBSConfig 读取 TestMain 准备好的 MinIO（S3 兼容）端点配置。
// TestMain 在 STORAGE_LOCK_OBS_ENDPOINT 未设置时会用 docker 自启 MinIO 并设置好 env。
func testOBSConfig(t *testing.T) (endpoint, accessKey, secretKey, bucket string) {
	t.Helper()
	endpoint = os.Getenv("STORAGE_LOCK_OBS_ENDPOINT")
	if endpoint == "" {
		t.Skip("STORAGE_LOCK_OBS_ENDPOINT not set and MinIO not available")
	}
	accessKey = os.Getenv("STORAGE_LOCK_OBS_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}
	secretKey = os.Getenv("STORAGE_LOCK_OBS_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin"
	}
	bucket = os.Getenv("STORAGE_LOCK_OBS_BUCKET")
	if bucket == "" {
		bucket = "storage-lock-test"
	}
	return endpoint, accessKey, secretKey, bucket
}

func newOBSMinioClient(t *testing.T, endpoint, accessKey, secretKey string) *minio.Client {
	t.Helper()
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	assert.Nil(t, err)
	return client
}

func ensureOBSBucket(t *testing.T, ctx context.Context, client *minio.Client, bucket string) {
	t.Helper()
	exists, err := client.BucketExists(ctx, bucket)
	assert.Nil(t, err)
	if !exists {
		assert.Nil(t, client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}))
	}
}

func TestNewOBSLockByClient(t *testing.T) {
	endpoint, accessKey, secretKey, bucket := testOBSConfig(t)
	ctx := context.Background()
	client := newOBSMinioClient(t, endpoint, accessKey, secretKey)
	ensureOBSBucket(t, ctx, client, bucket)

	factory, err := GetOBSLockFactoryByClient(ctx, client, bucket)
	assert.Nil(t, err)
	assert.NotNil(t, factory)

	// 缓存命中
	factory2, err := GetOBSLockFactoryByClient(ctx, client, bucket)
	assert.Nil(t, err)
	assert.Equal(t, factory, factory2)

	lock, err := NewOBSLockByClient(ctx, client, bucket, "test-obs-client-lock")
	assert.Nil(t, err)
	assert.NotNil(t, lock)
	assert.Nil(t, lock.Lock(ctx, "owner-1"))
	assert.Nil(t, lock.UnLock(ctx, "owner-1"))

	opts := storage_lock.NewStorageLockOptionsWithLockId("test-obs-client-lock-opt")
	lock2, err := NewOBSLockByClientWithOptions(ctx, client, bucket, opts)
	assert.Nil(t, err)
	assert.NotNil(t, lock2)
}

func TestNewOBSLockByEndpointWrappers(t *testing.T) {
	endpoint, accessKey, secretKey, bucket := testOBSConfig(t)
	ctx := context.Background()

	// 用 127.0.0.1 与 localhost 写法区分缓存 key，并带上 Region，覆盖 Region 分支
	regionEndpoint := endpoint
	if regionEndpoint == "localhost:9012" {
		regionEndpoint = "127.0.0.1:9012"
	} else {
		regionEndpoint = "127.0.0.1:9012"
	}
	config := &EndpointConfig{
		Endpoint:  regionEndpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		UseSSL:    false,
		Region:    "us-east-1",
		Bucket:    bucket,
	}

	factory, err := GetOBSLockFactoryByEndpoint(ctx, config)
	assert.Nil(t, err)
	assert.NotNil(t, factory)

	// 缓存命中
	factory2, err := GetOBSLockFactoryByEndpoint(ctx, config)
	assert.Nil(t, err)
	assert.Equal(t, factory, factory2)

	lock, err := NewOBSLockByEndpoint(ctx, config, "test-obs-endpoint-lock")
	assert.Nil(t, err)
	assert.NotNil(t, lock)

	opts := storage_lock.NewStorageLockOptionsWithLockId("test-obs-endpoint-lock-opt")
	lock2, err := NewOBSLockByEndpointWithOptions(ctx, config, opts)
	assert.Nil(t, err)
	assert.NotNil(t, lock2)
}

func TestOBSLock_StorageError(t *testing.T) {
	old := newOBSStorage
	newOBSStorage = func(ctx context.Context, options *obs_storage.OBSStorageOptions) (storage.Storage, error) {
		return nil, errors.New("boom")
	}
	defer func() { newOBSStorage = old }()

	ctx := context.Background()

	// 独立 client 避免命中成功用例的缓存（bean factory 按指针缓存）
	errClient, err := minio.New("127.0.0.1:9999", &minio.Options{
		Creds:  credentials.NewStaticV4("boom", "boom", ""),
		Secure: false,
	})
	assert.Nil(t, err)

	_, err = GetOBSLockFactoryByClient(ctx, errClient, "boom-bucket")
	assert.NotNil(t, err)
	_, err = NewOBSLockByClient(ctx, errClient, "boom-bucket", "x")
	assert.NotNil(t, err)
	_, err = NewOBSLockByClientWithOptions(ctx, errClient, "boom-bucket", storage_lock.NewStorageLockOptionsWithLockId("x"))
	assert.NotNil(t, err)

	// 独立 endpoint key，避免命中成功用例的缓存
	errConfig := &EndpointConfig{
		Endpoint:  "127.0.0.1:9999",
		AccessKey: "boom",
		SecretKey: "boom",
		Bucket:    "boom-bucket",
	}
	_, err = GetOBSLockFactoryByEndpoint(ctx, errConfig)
	assert.NotNil(t, err)
	_, err = NewOBSLockByEndpoint(ctx, errConfig, "x")
	assert.NotNil(t, err)
	_, err = NewOBSLockByEndpointWithOptions(ctx, errConfig, storage_lock.NewStorageLockOptionsWithLockId("x"))
	assert.NotNil(t, err)
}
