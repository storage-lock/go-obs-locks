# go-obs-locks

华为云 OBS 分布式锁便捷封装（S3 协议兼容）。

## 用法

```go
import obs_locks "github.com/storage-lock/go-obs-locks"

// 从端点创建
lock, err := obs_locks.NewOBSLockByEndpoint(ctx, &obs_locks.EndpointConfig{
    Endpoint:  "obs.cn-north-4.myhuaweicloud.com",
    AccessKey: "ak",
    SecretKey: "sk",
    UseSSL:    true,
    Bucket:    "my-bucket",
}, "my-lock")

// 从已有客户端创建
lock, err := obs_locks.NewOBSLockByClient(ctx, minioClient, "my-bucket", "my-lock")
```
