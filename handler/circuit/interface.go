package circuit

// Breaker 断路器接口
type Breaker interface {
	// GetAPIStatus 获取接口状态
	GetAPIStatus(key string) int
	// ChangeAPIStatus 修改接口状态
	ChangeAPIStatus(key string, status int)
}
