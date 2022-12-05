package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"
)

var askSleep = 1

// ByKey for sorting by key.
type ByKey []KeyValue

// Len for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// KeyValue map函数返回key value切片
type KeyValue struct {
	Key   string
	Value string
}

// ihash(key)
// 借助ihash函数%nreduce来选择对应的reduce
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// mapTask 处理字符串，然后输出到对应文件
func mapTask(reply *TaskReply, mapf func(string, string) []KeyValue) {
	filename := reply.Filename
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	content, err := ioutil.ReadAll(file) // 读取文件内容
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	file.Close()
	intermediate := mapf(filename, string(content))

	// 写入文件
	interArr := make([][]KeyValue, reply.NReduce)
	for i := 0; i < len(intermediate); i++ {
		index := ihash(intermediate[i].Key) % reply.NReduce // 找到字符串哈希后对应的reduce工作编号
		interArr[index] = append(interArr[index], intermediate[i])
	} // 将每个单词分别写入对应的array

	for i := 0; i < len(interArr); i++ {
		// FIXME 文件没有内容，跳过

		// 创建文件
		var oname = "mr-" + strconv.Itoa(reply.Index) + "-" + strconv.Itoa(i)

		//创建临时文件
		curDir, _ := os.Getwd()
		tmpFile, err := ioutil.TempFile(curDir, "tmp-map-")
		if err != nil {
			log.Fatal("cannot creat temporary file", err)
		}

		// 借助json写入临时文件
		enc := json.NewEncoder(tmpFile)
		for _, kv := range interArr[i] {
			err := enc.Encode(&kv)
			if err != nil {
				fmt.Println("error in writing map results", err)
			}
		}

		// 判断文件是否存在，不存在才可以重命名
		if _, err := os.Stat(oname); os.IsNotExist(err) {
			err = os.Rename(tmpFile.Name(), oname)
			if err != nil {
				fmt.Println("cannot rename tmp file", err)
			}
		} else {
			err := os.Remove(tmpFile.Name())
			if err != nil {
				return
			}
		}
	}

	// 所有工作做完，提醒coordinator
	args := TaskArgs{1, reply.Index}
	call("Coordinator.ListenTaskStatus", &args, reply)

	return
}

//para: 对应reduce编号，查询以num结尾的文件！
func reduceTask(reply *TaskReply, reducef func(string, []string) string) {
	//fmt.Println("reduce worker ", reply.Index, " is working")
	//创建临时文件
	curDir, _ := os.Getwd()
	tmpFile, _ := ioutil.TempFile(curDir, "tmp-reduce-")
	oname := reply.Filename

	// 读取目录下reduce对应的文件
	files, _ := ioutil.ReadDir("./")
	var kva []KeyValue
	for _, filename := range files {
		pattern := "mr-[0-9]*-" + strconv.Itoa(reply.Index)    // 匹配模式
		res, _ := regexp.MatchString(pattern, filename.Name()) // 获取匹配结果

		if res { // 文件存在
			// 读取所有kv对
			file, _ := os.Open(filename.Name())
			dec := json.NewDecoder(file)
			for {
				var kv KeyValue
				if err := dec.Decode(&kv); err != nil {
					break
				}
				kva = append(kva, kv)
			}
		}
	}
	// 获取reduce结果，写入临时文件
	sort.Sort(ByKey(kva))
	i := 0
	for i < len(kva) {
		j := i + 1
		for j < len(kva) && kva[i].Key == kva[j].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, kva[k].Value)
		}
		output := reducef(kva[i].Key, values)
		fmt.Fprintf(tmpFile, "%v %v\n", kva[i].Key, output)

		i = j
	}

	// 判断文件是否存在，不存在才可以重命名
	if _, err := os.Stat(oname); os.IsNotExist(err) {
		err = os.Rename(tmpFile.Name(), oname)
		if err != nil {
			fmt.Println("cannot rename tmp file", err)
		}
	} else {
		err := os.Remove(tmpFile.Name())
		if err != nil {
			return
		}
	}

	// 所有工作做完，提醒coordinator
	args := TaskArgs{2, reply.Index}
	call("Coordinator.ListenTaskStatus", &args, &reply)
	return

}

//ask-forTask
func askforTask(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {
	args := TaskArgs{}
	reply := TaskReply{}

	call("Coordinator.DeliverTask", &args, &reply)
	taskType := reply.TaskType
	switch {
	case taskType == 1:
		mapTask(&reply, mapf)

	case taskType == 2:
		reduceTask(&reply, reducef)

	default:
		return
	}
	return
}

// main/mrworker.go calls this function. 轮询master是否有任务
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {

	for true {
		askforTask(mapf, reducef)
		time.Sleep(time.Duration(askSleep) * time.Second)
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
// 给coordinator发送rpc请求，等待响应
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
