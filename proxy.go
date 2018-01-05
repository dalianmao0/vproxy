package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"hank.org/vproxy/conf"
)

var (
	vlist map[string]string
)

func getURIByName(name string) string {
	if v, ok := vlist[name]; ok {
		return v
	}
	fmt.Printf("Key %s NOT found. \n", name)
	return ""
}

// TODO: 优化转码逻辑：
// 1. 一个摄像头，最多开一个转码。
// 2. 每个摄像头下，增加一个客户端队列，管理连接过来的客户，转码后的流发送给队列中的所有客户。
func runFFMpeg(w http.ResponseWriter, r *http.Request, name string) {
	uri := getURIByName(name)
	if uri == "" {
		fmt.Println("NO such IPC found: key=", name)
		return
	}

	fmt.Println("runFFMpeg: ", name, uri)

	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Accept-Ranges", "bytes")

	cmd := exec.Command("ffmpeg",
		"-y",  // 默认自动覆盖输出文件，而不再询问确认
		"-re", // 以本地帧频读数据，主要用于模拟捕获设备
		"-rtsp_transport",
		"tcp",
		"-i",
		uri,
		// "-g 52", // 强制每第 52 帧作为关键帧
		"-vcodec",
		"copy",
		"-f",
		"mp4",
		"-movflags",
		"frag_keyframe", // <- for Chrome,
		// "frag_keyframe+empty_moov", // <- for Firefox
		"-reset_timestamps",
		"1",
		"-vsync",
		"1",
		"-flags",
		"global_header",
		"-bsf:v", // video bitstream filter
		"dump_extra",
		"-")
	printCommand(cmd)
	randomBytes := &bytes.Buffer{}
	cmd.Stdout = w

	// Start command asynchronously
	err := cmd.Start()
	printError(err)

	if _, err := w.Write(randomBytes.Bytes()); err != nil {
		// log.Println("unable to write output.")
		fmt.Println("unable to write output.")
	}

	// system blocked here until the transcoding finished
	cmd.Wait()

	// log.Println("leave ...")
	fmt.Println("Transcoding finished, FFMpeg exits.")
}

func printCommand(cmd *exec.Cmd) {
	fmt.Printf("==> Executing command: %s\n", strings.Join(cmd.Args, " "))
}

func printError(err error) {
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("==> Error: %s\n", err.Error()))
	}
}

func sendStream(w http.ResponseWriter, r *http.Request) {
	params, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		fmt.Println("No valid parameter: ", r.URL.RawQuery)
		return
	}

	runFFMpeg(w, r, params.Get("name"))
}

func loadConfig() {
	fmt.Print("Loading config ... ")

	alist, err := conf.Load("./config.json")
	if err != nil {
		fmt.Println("load config failed. ", err)
	}
	vlist = alist

	fmt.Println("done.")
}

func main() {
	// load the config
	loadConfig()

	// deal all static resource request
	http.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("./app"))))

	// deal all ip-camera request
	http.HandleFunc("/ipc/", sendStream)

	// start the web server
	fmt.Println("Proxy is going to listen on 9000...")
	err2 := http.ListenAndServe(":9000", nil)
	if err2 != nil {
		log.Fatal("ListenAndServe: ", err2)
	}
}
