package controlget

import (
	"net/http"
	"fmt"
	"log"
	//"path/filepath"
	"flag"
	//"io/ioutil"
	//"strings"
	//"os"
	//"context"
	"time"
	"net/url"
	"strings"
	"sync"
	"math/rand"

	 
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kubeinformers "k8s.io/client-go/informers"
	v1apps "k8s.io/client-go/informers/apps/v1"
	v1core "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	corelister "k8s.io/client-go/listers/core/v1"



)

const ( 
	watchdogPort = 8080
	namespace = "openfaas-fn"
)
var kubeconfig string
var masterURL string
//change to your apiGateway address
// var apiGateway = "http://172.16.252.163:31112/function/"
// var servicesilices []string
// var endpointsilices []string

type serverSetup struct {
	kubeClient             *kubernetes.Clientset
	kubeInformerFactory    kubeinformers.SharedInformerFactory
}
type customInformers struct {
	EndpointsInformer  v1core.EndpointsInformer
	DeploymentInformer v1apps.DeploymentInformer
				
}
// get kubeconfig
func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "","Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func Control(functionName string) (resp *http.Response, err error) {

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
	defaultResync := time.Minute * 5
	kubeInformerOpt := kubeinformers.WithNamespace(namespace)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClient, defaultResync, kubeInformerOpt)
	setup := serverSetup{
		kubeClient:             kubeClient,
		kubeInformerFactory:    kubeInformerFactory,
	}
	stopCh := make(chan struct{})
	defer close(stopCh)
	listers := startInformers(setup, stopCh)
	functionLookup := NewFunctionLookup(namespace, listers.EndpointsInformer.Lister())
	
	functionAddr, resolveErr := functionLookup.Resolve(functionName)
	if resolveErr != nil {
	// TODO: Should record the 404/not found error in Prometheus.
	log.Printf("resolver error: no endpoints for %s: %s\n", functionName, resolveErr.Error())
	}
	
	log.Printf("FunctionName: %s, ResolveAddr: %s", functionName, functionAddr)
	//urlStr := fmt.Sprintf("%s",&functionAddr)
	
	urlstr := functionAddr.String()
	resp, err = http.Get(urlstr)
	if err != nil {
		log.Fatalf("HTTP error: %s", err.Error())
	}
	return resp, err

	//services, err := kubeClient.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	//servicesilices = append(servicesilices, services.Spec.ClusterIP)

	// services, err := kubeClient.CoreV1().Services(namespace).Get(context.TODO(),functionName,metav1.GetOptions{})
	//pods, err := kubeClient.CoreV1().Pods(namespace).Get(context.TODO(),functionName,metav1.GetOptions{})
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	//endpointsilices = append(endpointsilices, pods.Status.PodIP)

	
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
	// if len(endpointsilices) == 0 {
	// 	resp, err := http.Get(apiGateway + functionName)
	// 	if err != nil {
	// 		fmt.Printf("err")
	// 	}
	// 	body, err := ioutil.ReadAll(resp.Body)
	// 	defer resp.Body.Close()
	// 	fmt.Printf("response is :%s", string(body))
	// 	return resp, err
	// } else {
	// 	urlStr := fmt.Sprintf("http://%s:%d", pods.Status.PodIP, watchdogPort)
	// 	resp, err := http.Get(urlStr)
	// 		//defer resp.Body.Close()
	// 	if err != nil {
	// 		log.Fatalf("HTTP error: %s", err.Error())
	// 	}
	// 	return resp, err
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
		

	// }
	// if len(endpointsilices) == 0 {
	// 	resp, err :=http.Get(apiGateway + functionaddress)

	// 	} else {
	// 	for i := range endpointsilices {
	// 		respc, errc := http.Get(endpointsilices[i])
	// 		//defer resp.Body.Close()
	// 		//body, err := ioutil.ReadAll(respc.Body)

	// 	}
}

func startInformers(setup serverSetup, stopCh <-chan struct{}) customInformers {
	kubeInformerFactory := setup.kubeInformerFactory
	

	deployments := kubeInformerFactory.Apps().V1().Deployments()
	go deployments.Informer().Run(stopCh)
	if ok := cache.WaitForNamedCacheSync("deployments", stopCh, deployments.Informer().HasSynced); !ok {
		log.Fatalf("failed to wait for cache to sync")
	}

	endpoints := kubeInformerFactory.Core().V1().Endpoints()
	go endpoints.Informer().Run(stopCh)
	if ok := cache.WaitForNamedCacheSync("endpoints", stopCh, endpoints.Informer().HasSynced); !ok {
		log.Fatalf("failed to wait for cache to sync")
	}


	return customInformers{
		EndpointsInformer:  endpoints,
		DeploymentInformer: deployments,
	}
}

func NewFunctionLookup(ns string, lister corelister.EndpointsLister) *FunctionLookup {
	return &FunctionLookup{
		DefaultNamespace: ns,
		EndpointLister:   lister,
		Listers:          map[string]corelister.EndpointsNamespaceLister{},
		lock:             sync.RWMutex{},
	}
}

type FunctionLookup struct {
	DefaultNamespace string
	EndpointLister   corelister.EndpointsLister
	Listers          map[string]corelister.EndpointsNamespaceLister
	lock sync.RWMutex
															
}

func (l *FunctionLookup) Resolve(name string) (url.URL, error) {
	functionName := name
	namespace := l.DefaultNamespace

	if strings.Contains(name, ".") {
		functionName = strings.TrimSuffix(name, "."+namespace)
	}

	nsEndpointLister := l.GetLister(namespace)

	if nsEndpointLister == nil {
		l.SetLister(namespace, l.EndpointLister.Endpoints(namespace))

		nsEndpointLister = l.GetLister(namespace)
	}

	svc, err := nsEndpointLister.Get(functionName)
	if err != nil {
		return url.URL{}, fmt.Errorf("error listing \"%s.%s\": %s", functionName, namespace, err.Error())
	}

	if len(svc.Subsets) == 0 {
		return url.URL{}, fmt.Errorf("no subsets available for \"%s.%s\"", functionName, namespace)
	}

	all := len(svc.Subsets[0].Addresses)
	if len(svc.Subsets[0].Addresses) == 0 {
		return url.URL{}, fmt.Errorf("no addresses in subset for \"%s.%s\"", functionName, namespace)
	}

	target := rand.Intn(all)

	serviceIP := svc.Subsets[0].Addresses[target].IP

	urlStr := fmt.Sprintf("http://%s:%d", serviceIP, watchdogPort)

	urlRes, err := url.Parse(urlStr)
	if err != nil {
		return url.URL{}, err
	}

	log.Printf("[Call k8s/proxy.go Resolve] name: %s, url %s", name, urlStr)

	return *urlRes, nil
}

func (f *FunctionLookup) GetLister(ns string) corelister.EndpointsNamespaceLister {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.Listers[ns]
}

func (f *FunctionLookup) SetLister(ns string, lister corelister.EndpointsNamespaceLister) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.Listers[ns] = lister
}
