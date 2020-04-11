package main

import (
	"flag"
	"github.com/golang/glog"
)

//  初始化
func init() {
	//  直接初始化，主要使服务器启动后自己直接加载，并不用命令行执行对应的参数
	flag.Set("alsologtostderr", "true") // 日志写入文件的同时，输出到stderr
	//flag.Set("log_dir", "./log")        // 日志文件保存目录
	flag.Set("v", "3")                  // 配置V输出的等级。
	flag.Parse()
}

// 主函数
func main() {
	//  使用

	// 退出时调用，确保日志写入文件中
	// defer glog.Flush()

	glog.Info("hello, glog")
	glog.Warning("warning glog")
	glog.Error("error glog")

	//  直接刷新到文件中
	glog.Flush()
	return
}

