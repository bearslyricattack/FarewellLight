package FarewellLight

import (
	"FarewellLight/data"
	"FarewellLight/handler/circuit"
	"FarewellLight/handler/merge"
	"errors"
)

type StrategyInterface interface {
	Do(key string, strategy int, fn func() (interface{}, error)) (v interface{}, err error, shared bool)
}

type ApiStatus struct {
	ApiName    string
	IsPaused   int
	ErrCount   int64
	TotalCount int64

	AccessLast int64 // last access timestamp of api
	RoundLast  int64 // start timestamp of this round
}

// StrategyClient 请求策略client
type StrategyClient struct {

	//数据源client
	data data.BaseClient

	//接口内容的格式
	status ApiStatus

	//熔断器client
	circuit circuit.Breaker

	//合并器client
	merge merge.Combine

	//合并请求的熔断策略
	specialStrategy int
}

func (s *StrategyClient) Do(key string, strategy int, fn func() (interface{}, error)) (v interface{}, err error, shared bool) {
	//首先检查接口是否需要熔断
	if s.circuit != nil {
		//如果需要熔断，检查接口状态，当前为2代表已经熔断，直接返回
		res := s.circuit.GetAPIStatus(key)
		//如果当前是错误，直接返回
		if res == 2 {
			s.circuit.ChangeAPIStatus(key, 0)
			return nil, err, false
		}
	}
	//如果不需要熔断，那么判断接口是否需要合并
	if s.merge == nil  {
		//更新接口状态
		value,err := fn()
		if errors.Is(err,CbkError{}){
			s.circuit.ChangeAPIStatus(key,0)
		}else {
			s.circuit.ChangeAPIStatus(key,1)
		}
		return value,err,true
	}
	//1 代表合并的接口不处理熔断策略
	if s.specialStrategy == 1 {
		s.merge.Do(key,0,fn)
		//更新接口状态
		value,err := fn()
		return value,err,true
	}
	//2 代表正常处理
	if s.specialStrategy == 2 {
		value,err, _ := s.merge.Do(key,0,fn)
		//更新接口状态
		if errors.Is(err,CbkError{}){
			s.circuit.ChangeAPIStatus(key,0)
		}else {
			s.circuit.ChangeAPIStatus(key,1)
		}
		return value,err,true
	}
	//兜底
	value,err := fn()
	return value,err,true

}

type CbkError struct {
	error
	Msg string
}
