package merge

// Combine 合并器接口
type Combine interface {
	Do(key string, strategy int, fn func() (interface{}, error)) (v interface{}, err error, shared bool)
}
