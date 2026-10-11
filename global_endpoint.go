package obs_locks

import (
	"context"

	"github.com/minio/minio-go/v7"
	obs_storage "github.com/storage-lock/go-obs-storage"
	storage_lock "github.com/storage-lock/go-storage-lock"
	storage_lock_factory "github.com/storage-lock/go-storage-lock-factory"
)

// obsEndpointBeanFactory 缓存基于端点的锁工厂
var obsEndpointBeanFactory = storage_lock_factory.NewStorageLockFactoryBeanFactory[string, *minio.Client]()

// EndpointConfig OBS 端点配置
type EndpointConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Region    string
	Bucket    string
}

// NewOBSLockByEndpoint 从端点参数创建分布式锁
func NewOBSLockByEndpoint(ctx context.Context, config *EndpointConfig, lockId string) (*storage_lock.StorageLock, error) {
	factory, err := GetOBSLockFactoryByEndpoint(ctx, config)
	if err != nil {
		return nil, err
	}
	return factory.CreateLock(lockId)
}

// NewOBSLockByEndpointWithOptions 从端点参数和锁选项创建分布式锁
func NewOBSLockByEndpointWithOptions(ctx context.Context, config *EndpointConfig, options *storage_lock.StorageLockOptions) (*storage_lock.StorageLock, error) {
	factory, err := GetOBSLockFactoryByEndpoint(ctx, config)
	if err != nil {
		return nil, err
	}
	return factory.CreateLockWithOptions(options)
}

// GetOBSLockFactoryByEndpoint 获取基于端点的锁工厂（单例模式，endpoint 作缓存 key）
func GetOBSLockFactoryByEndpoint(ctx context.Context, config *EndpointConfig) (*storage_lock_factory.StorageLockFactory[*minio.Client], error) {
	// 用 endpoint+bucket 作为缓存 key
	cacheKey := config.Endpoint + "/" + config.Bucket

	return obsEndpointBeanFactory.GetOrInit(ctx, cacheKey, func(ctx context.Context) (*storage_lock_factory.StorageLockFactory[*minio.Client], error) {
		cm := obs_storage.NewOBSConnectionManagerFromEndpoint(config.Endpoint, config.AccessKey, config.SecretKey, config.UseSSL)
		if config.Region != "" {
			cm.Region = config.Region
		}
		options := obs_storage.NewOBSStorageOptions().
			SetConnectionManager(cm).
			SetBucketName(config.Bucket).
			SetTimeProvider(&obs_storage.LocalTimeProvider{})
		storage, err := newOBSStorage(ctx, options)
		if err != nil {
			return nil, err
		}
		return storage_lock_factory.NewStorageLockFactory[*minio.Client](storage, nil), nil
	})
}
