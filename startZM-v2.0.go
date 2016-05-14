//pscp -pw table1 -P 23 d:\src\*.go t1@ccnu.sip.300y.com.cn:/home/t1/

package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
	//"os/exec"
	//"syscall"
	"encoding/json"
	"io/ioutil"
)

type StreamInfo struct { /*定义结构体*/
	Id int //流的ID号
	//proc *os.Process	//流的子进程Process
	Name string //流的名称与rtmp name
	Rtsp string //流的rtsp输入串
}

type StreamProc struct {
	Id   int         //流的ID号
	Name string      //流的名称
	Proc *os.Process //Pid的通道
}

var wg sync.WaitGroup

func main() {
	flags()                                                        //解析命令行参数
	var stream map[string]StreamInfo = make(map[string]StreamInfo) //MAP变量
	var sm chan StreamInfo                                         //定义通道，用于向子进程传递数据
	var sp chan StreamProc                                         //定义子处理进程 发送的通道
	var stream1 map[string]string

	sm = make(chan StreamInfo, 1)
	sp = make(chan StreamProc, 1)

	stream1, err := rJson("stream.json") //读取配置文件并赋值给map
	if err != nil {
		log.Println("readFile Fail:", err.Error())
	}
	var streamcount int
	streamcount = len(stream1) //取得要转码流的数量
	log.Println("stream1 number:", streamcount)

	cpucount := runtime.NumCPU()
	log.Println("cpu core num:", cpucount)
	time.Sleep(time.Second * 1) //休眠1秒
	runtime.GOMAXPROCS(cpucount)

	wg.Add(streamcount + 1)
	//os.Stdin.Close();
	//os.Stdout.Close();
	//os.Stderr.Close();
	for k, v := range stream1 {
		//log.Println(k, stream1[k])
		stream[k] = StreamInfo{0, k, v}
		sm <- StreamInfo{0, k, v}
		go ForkFF(sm, sp) //启动进程调用ffmpeg
	}
	//go jk(sm, sp, stream)
	jk(sm, sp, stream)
	wg.Wait()
	log.Println("DONE!")
}
func jk(sm chan<- StreamInfo, sp <-chan StreamProc, stream map[string]StreamInfo) {
	defer wg.Done() //重要
	var i int = 0
	var _sp StreamProc
	var _sm StreamInfo
	for _sp = range sp {
		i++
		if i == 50 {
			err := _sp.Proc.Kill()
			/*if err := p.Signal(os.Kill); err != nil { //也可以发送一个干掉他的信号
			    log.Println(err)
			}*/
			log.Println("kill Pid:", _sp.Proc.Pid, "\t err:", err)
			err = _sp.Proc.Release()
			log.Println("Release Pid:", _sp.Proc.Pid, "\t err:", err)
			stream[_sp.Name] = StreamInfo{0, _sp.Name, "heeh"}
			_sm = stream[_sp.Name]
		}
		//p = <- sp;
		log.Println("sp.Name:", _sp.Name, "\t sp.Proc.Pid:", _sp.Proc.Pid, "\t sp.Id:", _sp.Id)
		_sm = stream[_sp.Name]
		sm <- _sm //发送通道，让子处理进程继续循环处理上一次的程序
	}
}
func ForkFF(sm <-chan StreamInfo, sp chan<- StreamProc) {
	var _sm StreamInfo
	var ok bool
	defer func() {
		log.Println("defer:")
	}()
	defer wg.Done() //重要
	log.Println("start fork child process:")
	var proc *os.Process
	var procattr *os.ProcAttr
	//procAttr.Files = []*os.File{nil, os.Stdout, os.Stderr}
	var err error
	var command string
	command = "/usr/local/bin/ffmpeg"

	Stdnull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	procattr = &os.ProcAttr{Files: []*os.File{Stdnull, Stdnull, Stdnull}} //可以使用，不进行交互
	//procattr=&os.ProcAttr{Files:[]*os.File{nil,nil,nil}} 				//不能使用
	//procattr = &os.ProcAttr{ Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}	}
	var args []string
	for {
		_sm, ok = <-sm //阻塞，通道接收
		if ok == false {
			return
		}
		/*args = []string{command, "-y", "-fflags", "+genpts", "-rtsp_flags", "prefer_tcp", "-i",
		_sm.Rtsp, "-flags", "+global_header", "-c:v", "libx264", "-thread_type", "slice",
		"-crf", "30", "-maxrate", "512k", "-profile:v", "main", "-level", "3.1", "-g", "12",
		"-pix_fmt", "yuv420p", "-strict", "experimental", "-an", "-f", "flv", "-rtmp_live", "live",
		"-rtmp_buffer", "500", "rtmp://127.0.0.1:9901/live/" + _sm.Name}*/
		args = []string{command, "-y", "-i", _sm.Rtsp, "-vcodec", "copy", "-an",
			"-f", "flv", "rtmp://127.0.0.1:1935/" + _sm.Name}
		//log.Println("command:", command)
		//log.Println("args:", args)
		proc, err = os.StartProcess(command, args, procattr)
		//proc,err = os.StartProcess("/bin/ls",[]string{"/bin/ls","/tmp"},procattr);	//必须要加完整的执行路径
		if err != nil {
			log.Println("startProcess faild.\n")
		} else {
			log.Println("fork child,pid:", proc.Pid, "\tstream[k].pid", 0)
			//stream[k].proc=proc;	//不能使用，估计因为map类型只能修改键值，不能修改键值的元素，但是可以取键值的元素如stream[k].proc.Pid
			//stream[k] = StreamInfo{0,<-proc,stream[k].Name,stream[k].Rtsp};		//发送通道
			sp <- StreamProc{_sm.Id, _sm.Name, proc} //发送通道
			//stream[k].proc <- proc		//发送通道
		}
		var exitstatus *os.ProcessState
		exitstatus, err = proc.Wait()
		if err != nil {
			log.Println("wait process faild.\n")
		}
		log.Println("exit status:", exitstatus)
	}
}

func rJson(filename string) (map[string]string, error) {
	/*#cat stream.json
	  {
	  "ch5":"rtsp://8888:yc888888@60.161.131.180:554/h264/ch5/sub/av_stream",
	  "ch6":"rtsp://8888:yc888888@60.161.131.180:554/h264/ch6/sub/av_stream",
	  "ch7":"rtsp://8888:yc888888@60.161.131.180:554/h264/ch7/sub/av_stream",
	  "ch8":"rtsp://8888:yc888888@60.161.131.180:554/h264/ch8/sub/av_stream"
	  }
	*/
	var streamconf = map[string]string{}
	jsonstr, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println("ReadFile:", err.Error())
		return nil, err
	}
	if err := json.Unmarshal(jsonstr, &streamconf); err != nil {
		log.Println("Unmarshal:", err.Error())
		return nil, err
	}
	/*log.Println("read json streamconf start:")
		for k, v := range streamconf {
			log.Println(k, v)
	        }
	      log.Println("read json streamconf end!")*/
	return streamconf, nil
}

func flags() {
	//取命令行参数并解析
	zmnum := flag.Int("zmnum", 20, "最大的转码数量")
	corenum := flag.Int("corenum", 1, "CPU核心数量")
	v := flag.Bool("v", false, "显示版本信息")
	flag.Parse()
	if *v == true {
		log.Println("print verison!")
		os.Exit(100)
	}
	log.Println("分析命令行输入参数  Args \t zmnum:", *zmnum, "; corenum:", *corenum)

}
