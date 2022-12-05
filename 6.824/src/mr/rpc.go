package mr

//
// ＲＰＣ定义
// 记得所有命名首字母大写

import "os"
import "strconv"

type TaskReply struct {
	TaskType int    // 记录任务类型
	Index    int    // 记录任务编号
	Filename string // 记录map worker读取的文件名 or reduce写的文件名
	NReduce  int    // 用于map阶段
}

type TaskArgs struct {
	TaskType int // 记录任务类型
	Index    int // 记录任务编号
}

// 在这里引入RPC声明

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
