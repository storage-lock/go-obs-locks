package obs_locks

import (
	"context"

	"github.com/minio/minio-go/v7"
	obs_storage "github.com/storage-lock/go-obs-storage"
	"github.com/storage-lock/go-storage"
	storage_lock "github.com/storage-lock/go-storage-lock"
	storage_lock_factory "github.com/storage-lock/go-storage-lock-factory"
)

// obsClientBeanFactory 缓存基于 OBS 客户端的锁工厂
var obsClientBeanFactory = storage_lock_factory.NewStorageLockFactoryBeanFactory[*minio.Client, *minio.Client]()

// newOBSStorage 指向 obs_storage.NewOBSStorage，包级变量以便单测注入构造失败，
// 覆盖 GetOrInit initFunc 的错误分支。client/endpoint 双入口共用。
var newOBSStorage = func(ctx context.Context, options *obs_storage.OBSStorageOptions) (storage.Storage, error) {
	return obs_storage.NewOBSStorage(ctx, options)
}

// NewOBSLockByClient 从已有 OBS 客户端创建分布式锁
func NewOBSLockByClient(ctx context.Context, client *minio.Client, bucketName string, lockId string) (*storage_lock.StorageLock, error) {
	factory, err := GetOBSLockFactoryByClient(ctx, client, bucketName)
	if err != nil {
		return nil, err
	}
	return factory.CreateLock(lockId)
}

// NewOBSLockByClientWithOptions 从已有 OBS 客户端和锁选项创建分布式锁
func NewOBSLockByClientWithOptions(ctx context.Context, client *minio.Client, bucketName string, options *storage_lock.StorageLockOptions) (*storage_lock.StorageLock, error) {
	factory, err := GetOBSLockFactoryByClient(ctx, client, bucketName)
	if err != nil {
		return nil, err
	}
	return factory.CreateLockWithOptions(options)
}

// GetOBSLockFactoryByClient 获取基于 OBS 客户端的锁工厂（单例模式）
func GetOBSLockFactoryByClient(ctx context.Context, client *minio.Client, bucketName string) (*storage_lock_factory.StorageLockFactory[*minio.Client], error) {
	return obsClientBeanFactory.GetOrInit(ctx, client, func(ctx context.Context) (*storage_lock_factory.StorageLockFactory[*minio.Client], error) {
		cm := obs_storage.NewOBSConnectionManagerFromClient(client)
		options := obs_storage.NewOBSStorageOptions().
			SetConnectionManager(cm).
			SetBucketName(bucketName).
			SetTimeProvider(&obs_storage.LocalTimeProvider{})
		storage, err := newOBSStorage(ctx, options)
		if err != nil {
			return nil, err
		}
		return storage_lock_factory.NewStorageLockFactory[*minio.Client](storage, nil), nil
	})
}
