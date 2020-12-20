// 实现了一个最简单的协程池
// 线程/写成池有多种实现方式，可以
// 1. 通过channel发送数据
// 2. 通过chanel发送任务（task）
// 3. 自行指定worker（当前采用的方式）
// 后续如果需要优化任务分配方式，则需要重新此实现（包括worker）即可
// NOTE: 注意当前实现没有处理worker崩溃、需要重启等问题
package routingpool

import (
	"context"
	"sync"
)

type SimpleRoutingPool struct {
	wg sync.WaitGroup

	ctx      context.Context
	size     uint32
	workerFn func(context.Context)
}

func NewSimpleRoutingPool(ctx context.Context, size uint32, workerFn func(context.Context)) RoutingPool {
	return &SimpleRoutingPool{
		ctx:      ctx,
		size:     size,
		workerFn: workerFn,
	}
}

func (s *SimpleRoutingPool) Start() error {
	var i uint32
	for ; i != s.size; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.workerFn(s.ctx)
		}()
	}
	return nil
}

func (s *SimpleRoutingPool) Stop() {
	s.wg.Wait()
}
