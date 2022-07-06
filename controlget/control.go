package controlget

import (
	"net/http"
	"fmt"
	"log"
	//"path/filepath"
	"flag"
	"io/ioutil"
	//"strings"
	//"os"
	"context"
	 
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"



)

const ( watchdogPort = 8080
	namespace = "openfaas-fn"
)
var kubeconfig string
var masterURL string
var apiGateway = "http://172.16.252.163:31112/function/"
var servicesilices []string

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "","Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func Control(functionName string) (resp *http.Response, err error) {

	//functionaddress := "/function/test-4"

	//var endpointsilices []string
	
	//var respslices []*http.Response

	//functionName := getServiceName(functionaddress)

	log.Printf("function name is: %s \n", functionName)

	// readConfig := config.ReadConfig{}
	// osEnv := providertypes.OsEnv{}
	// config, err := readConfig.Read(osEnv)

	

	// config.DefaultFunctionNamespace = namespace
	// var config string
	// var URL string

	// flag.StringVar(&config, "kubeconfig", "","Path to a kubeconfig. Only required if out-of-cluster.")
	// flag.StringVar(&URL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	
	// if home := homeDir(); home != "" {
	// 	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	// } else {
	// 	//kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	// 	flag.StringVar(kubeconfig, "kubeconfig", "","Path to a kubeconfig. Only required if out-of-cluster.")
	// 	log.Printf("kubeconfig do not exist")
	// }
	flag.Parse()
	
	clientCmdConfig, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(clientCmdConfig)
	if err != nil {
		log.Fatalf("Error building Kubernetes clientset: %s", err.Error())
	}

	//services, err := kubeClient.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	services, err := kubeClient.CoreV1().Services(namespace).Get(context.TODO(),functionName,metav1.GetOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}
	servicesilices = append(servicesilices, services.Spec.ClusterIP)
	
	// for _, s := range services.Items {
	// 	if strings.Contains(s.Name, functionName) {
	// 		fmt.Printf("Name: %v Cluster IP: %v\n", s.Name, s.Spec.ClusterIP)
	// 		servicesilices = append(servicesilices, s.Spec.ClusterIP)
	// 	}

	// }
	// pods, err := kubeClient.CoreV1().Pods(config.DefaultFunctionNamespace).List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// for _, v := range pods.Items {
	// 	if strings.Contains(v.Name, functionName) {
	// 		fmt.Printf("Name: %v IP: %v\n", v.Name, v.Status.PodIP)
	// 		endpointsilices = append(endpointsilices, v.Status.PodIP)
	// 	}

	// }
	if len(servicesilices) == 0 {
		resp, err := http.Get(apiGateway + functionName)
		if err != nil {
			fmt.Printf("err")
		}
		body, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		fmt.Printf("response is :%s", string(body))
		return resp, err
	} else {
		urlStr := fmt.Sprintf("http://%s:%d", services.Spec.ClusterIP, watchdogPort)
		resp, err := http.Get(urlStr)
			//defer resp.Body.Close()
		if err != nil {
			log.Fatalf("HTTP error: %s", err.Error())
		}
		return resp, err
		// for i := range servicesilices {
		// 	endpointIP := servicesilices[i]
		// 	urlStr := fmt.Sprintf("http://%s:%d", endpointIP, watchdogPort)
		// 	resp, err := http.Get(urlStr)
		// 	//defer resp.Body.Close()
		// 	if err != nil {
		// 		//fmt.Printf(err.Error())
		// 		log.Fatalf("HTTP error: %s", err.Error())
		// 	}
		// 	// body, err := ioutil.ReadAll(resp.Body)
		// 	// fmt.Printf("response is :%s \n", string(body))
		// 	//respslices = append(respslices, resp)
		// 	return resp, err
			
		// }
		

	}
	// if len(endpointsilices) == 0 {
	// 	resp, err :=http.Get(apiGateway + functionaddress)

	// 	} else {
	// 	for i := range endpointsilices {
	// 		respc, errc := http.Get(endpointsilices[i])
	// 		//defer resp.Body.Close()
	// 		//body, err := ioutil.ReadAll(respc.Body)

	// 	}
}

// func homeDir() string {
// 	if h := os.Getenv("HOME"); h != "" {
// 		return h
// 	}
// 	return os.Getenv("USERPROFILE") // windows
// }