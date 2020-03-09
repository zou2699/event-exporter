/*
@Time : 2020/3/6 14:29
@Author : Tux
@Description :
*/

package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// addr the Prometheus metrics listen on.
var addr = flag.String("listen-address", ":9001", "The address to listen on for HTTP requests.")
// 0: just warning + unknown  1: All
var eventLevel = flag.Int("event-level", 0, "event type 0: just warning and unknown  1: All")

func main() {
	var wg sync.WaitGroup

	clientset := loadConfig()
	// sharedInformerFactory for all namespaces
	sharedInformer := informers.NewSharedInformerFactory(clientset, 30*time.Minute)
	// 获取 event informer
	eventInformer := sharedInformer.Core().V1().Events()

	// 实例化 eventStore
	eventStore := NewEventStore(clientset, eventInformer)
	// 信号handler
	stopCh := sigHandler()

	// 启动 prometheus
	go func() {
		glog.Info("starting prometheus metrics")
		http.Handle("/metrics", promhttp.Handler())
		glog.Warning(http.ListenAndServe(*addr, nil))
	}()

	// 启动 eventStore
	wg.Add(1)
	go func() {
		defer wg.Done()
		eventStore.Run(stopCh)
	}()

	// Startup the sharedInformer
	glog.Infof("Starting shared Informer")
	sharedInformer.Start(stopCh)

	wg.Wait()
	glog.Warningf("Exiting main")
}

func sigHandler() <-chan struct{} {
	stopCh := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c,
			syscall.SIGINT,  // Ctrl+C
			syscall.SIGTERM, // Termination Request
			syscall.SIGSEGV, // FullDerp
			syscall.SIGABRT, // Abnormal termination
			syscall.SIGILL,  // illegal instruction
			syscall.SIGFPE) // floating point - this is why we can't have nice things
		sig := <-c
		glog.Warningf("Signal (%v) Detected, Shutting Down", sig)
		close(stopCh)
	}()
	return stopCh
}

// loadConfig will parse input + config file and return a clientset
func loadConfig() (clientset kubernetes.Interface) {
	var kubeconfig *string
	var config *rest.Config
	var err error
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "(optional) absolute path to the kubeconfig file")
	}
	flag.Parse()

	if len(*kubeconfig) > 0 {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		panic(err)
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return
}
