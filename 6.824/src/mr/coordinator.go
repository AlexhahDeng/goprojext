package mr

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
)

var rw sync.RWMutex
var sleepTime = 10

type MapTask struct {
	filename string // map任务文件名
	status   int    // 状态 0--incomplete, 1--complete, 2--runtime
}

type ReduceTask struct {
	num    int // reduce任务编号
	status int // 状态 0--incomplete, 1--complete, 2--runtime
}

// Coordinator  结构体
type Coordinator struct {
	nReduce       int // number of reduce tasks
	totMapTask    int // the number of map tasks
	comMapTask    int // the number of completed map tasks, at last should equal to totMapTask
	comReduceTask int // number of completed reduce tasks, at last should equal to nReduce

	mapList    []MapTask    // record all map tasks
	reduceList []ReduceTask // recode all reduce tasks

}

// 跟踪map任务是否超时
func (c *Coordinator) trackMapTask(index int) {
	// 睡10s再判断worker工作是否干完
	time.Sleep(time.Duration(sleepTime) * time.Second)

	// 直接看对应map task的状态是否改变
	rw.Lock()
	if c.mapList[index].status != 1 {
		// 无论是什么状态都改为，没完成
		//fmt.Println("map task:", index, " incompleted!")
		c.mapList[index].status = 0
	}
	rw.Unlock()
	return
}

// 跟踪reduce任务是否超时
func (c *Coordinator) trackReduceTask(index int) {
	// 睡10s再判断worker工作是否干完
	time.Sleep(time.Duration(sleepTime) * time.Second)

	// 直接看对应reduce task的状态是否改变
	rw.Lock()
	if c.reduceList[index].status != 1 {
		//fmt.Println("reduce task:", index, " incompleted!")
		c.reduceList[index].status = 0
	}
	rw.Unlock()
	return
}

// DeliverTask 分发任务
func (c *Coordinator) DeliverTask(args *TaskArgs, reply *TaskReply) error {
	reply.NReduce = c.nReduce
	// 判断map任务是否做完
	switch {
	case c.comMapTask < c.totMapTask:
		rw.Lock()
		defer rw.Unlock()
		for index, mapItem := range c.mapList {
			if mapItem.status == 0 {
				// 确保其他人不会再继续操作
				c.mapList[index].status = 2

				reply.Index = index
				reply.Filename = mapItem.filename // 需要读取的文件名
				reply.TaskType = 1

				go c.trackMapTask(index) // 分发任务以后开始记时
				return nil
			}
		}
	case c.comMapTask == c.totMapTask:
		rw.Lock()
		defer rw.Unlock()
		for index, reduceItem := range c.reduceList {
			if reduceItem.status == 0 {
				// 确保其他人不会再继续操作
				c.reduceList[index].status = 2

				reply.Index = index
				reply.TaskType = 2
				reply.Filename = "mr-out-" + strconv.Itoa(index)

				go c.trackReduceTask(index)
				return nil
			}
		}
	default:
		reply.TaskType = -1 // 不分配任何任务
		return errors.New("invalid comMapTask")
	}
	return nil
}

// ListenTaskStatus 接收worker请求，任务完成标识状态为1，以及修改已完成任务数量
func (c *Coordinator) ListenTaskStatus(args *TaskArgs, reply *TaskReply) error {

	switch {
	case args.TaskType == 1:
		rw.Lock()
		if c.mapList[args.Index].status != 1 {
			//fmt.Println("map task ", args.Index, " has done")
			c.mapList[args.Index].status = 1
			c.comMapTask += 1
		} // 如果任务已经完成，就不再处理==>失效的worker无视他
		rw.Unlock()

	case args.TaskType == 2:
		rw.Lock()
		if c.reduceList[args.Index].status != 1 {
			//fmt.Println("reduce task ", args.Index, " has done")
			c.reduceList[args.Index].status = 1
			c.comReduceTask += 1
		}
		rw.Unlock()
	}

	return nil
}

// 开启一个线程监听worker.go发出的RPC请求
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// Done mrcoordinator.go周期性调用Done()来判断整个任务是否结束
func (c *Coordinator) Done() bool {
	ret := false
	rw.RLock()
	if c.comReduceTask == c.nReduce {
		ret = true
	}
	rw.RUnlock()

	return ret
}

// MakeCoordinator /*
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	// initialization
	c := Coordinator{}
	c.nReduce = nReduce
	c.totMapTask = len(files)
	c.comMapTask = 0
	c.comReduceTask = 0

	//初始化map任务
	c.mapList = make([]MapTask, c.totMapTask)
	for i := 0; i < c.totMapTask; i++ {
		c.mapList[i] = MapTask{files[i], 0}
	}

	// 初始化reduce任务
	c.reduceList = make([]ReduceTask, nReduce)
	for i := 0; i < nReduce; i++ {
		c.reduceList[i] = ReduceTask{i, 0}
	}

	c.server()
	return &c
}
