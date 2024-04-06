package circuit

import (
	"FarewellLight"
	"FarewellLight/data"
	"context"
	"sync"
	"time"
)

// BreakerImplement 断路器的实现结构体
type BreakerImplement struct {

	//数据源
	data data.BaseClient

	//锁
	lock sync.RWMutex

	//最小阈值
	minCheck int64

	//错误比率
	cbkErrRate float64

	//恢复周期
	recoverInterval time.Duration

	//循环周期
	roundInterval time.Duration
}

// NewBreaker 初始化方法
func NewBreaker() *BreakerImplement {
	return &BreakerImplement{}
}

// GetAPIStatus 获取服务状态
func (b *BreakerImplement) GetAPIStatus(key string) int {

	res, err := b.data.Get(context.Background(), key)

	//如果没找到 直接返回true
	if !err {
		return 1
	}
	//接口定义
	req := res.(FarewellLight.ApiStatus)
	return req.IsPaused
}

// ChangeAPIStatus 新增调用次数,并修改接口状态
func (b *BreakerImplement) ChangeAPIStatus(key string, status int) {

	var req FarewellLight.ApiStatus
	//获取当前时间戳
	nowTime := time.Now().UnixNano()

	//根据key查询当前接口
	res, err := b.data.Get(context.Background(), key)

	if !err {
		//如果没找到 新增一个
		req = FarewellLight.ApiStatus{
			ApiName: key, IsPaused: 1, ErrCount: 0, TotalCount: 1, AccessLast: nowTime, RoundLast: nowTime,
		}
	} else {
		//如果找到了，进行断言
		req = res.(FarewellLight.ApiStatus)
	}
	//修改状态
	req.TotalCount += 1
	req.AccessLast = nowTime

	//如果已经大于一个周期，那么重置周期
	if Abs64(nowTime-req.AccessLast) > int64(b.roundInterval) {
		req.TotalCount = 0
		req.ErrCount = 0
		req.RoundLast = nowTime
	}
	// 0代表这次请求失败了，失败请求加一
	if status == 0 {
		req.ErrCount += 1
	}

	//修改接口状态
	//如果此时接口处于失败状态，那么进行一次放行
	if req.IsPaused == 2 {
		req.IsPaused = 3
	}
	//如果此时接口处于一次放行状态，那么检查结果
	if req.IsPaused == 3 {
		//唯一的一次放行成功了
		if status == 1 {
			req.IsPaused = 1
		} else {
			req.IsPaused = 2
		}
	}

	//如果此时接口处于成功状态，那么判断是否应该进入失败
	if req.IsPaused == 1 {
		latency := Abs64(nowTime - req.AccessLast)
		//判断是否在周期内,判断请求的总数，如果很少就不用管了,判断错误的阈值，到达就修改
		if latency < int64(b.recoverInterval) && req.TotalCount > b.minCheck && float64(req.ErrCount)/float64(req.TotalCount) > b.cbkErrRate {
			req.IsPaused = 2
		}
	}

	b.data.Set(context.Background(), key, req, 0)
	return
}

func Abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
