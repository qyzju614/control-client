package controlget

import (
	"fmt"
	"log"
	"net/http"

	//"path/filepath"
	//"flag"
	//"io/ioutil"
	//"strings"
	//"os"
	//"context"
	//"time"
	//"net/url"
	"math/rand"
	"strings"
	"sync"

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/tools/clientcmd"
	//kubeinformers "k8s.io/client-go/informers"
	//v1apps "k8s.io/client-go/informers/apps/v1"
	//v1core "k8s.io/client-go/informers/core/v1"
	//"k8s.io/client-go/tools/cache"
	kubeovnlist "github.com/kubeovn/kube-ovn/pkg/client/listers/kubeovn/v1"
	corelister "k8s.io/client-go/listers/core/v1"
)

const (
	watchdogPort = 8080
)

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
	lock             sync.RWMutex
}

type VipFunctionLookup struct {
	VipLister kubeovnlist.VipLister
}

func NewVipLookup(ns string, lister kubeovnlist.VipLister) *VipFunctionLookup {
	return &VipFunctionLookup{
		VipLister: lister,
	}
}

func (l *VipFunctionLookup) ResolveVip(name string) (resp *http.Response, err error) {
	vipname := "vip-" + name
	fakeip, err := l.VipLister.Get(vipname)
	if err != nil {
		log.Printf("failed to get static vip:, %v", err)
	}

	log.Printf("svc %s fakeip is %s", fakeip.Name, fakeip.Spec.V4ip)
	urlStr := fmt.Sprintf("http://%s:%d", fakeip.Spec.V4ip, watchdogPort)

	log.Printf("[ResolveVip] name: %s, url %s", name, urlStr)

	urlresp, err := http.Get(urlStr)
	if err != nil {
		log.Printf(err.Error())
	}
	return urlresp, err

}

func (l *FunctionLookup) Resolve(name string) (resp *http.Response, err error) {
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
		log.Printf("error listing \"%s.%s\": %s", functionName, namespace, err.Error())
	}

	if len(svc.Subsets) == 0 {
		log.Printf("no subsets available for \"%s.%s\"", functionName, namespace)
	}

	all := len(svc.Subsets[0].Addresses)
	if len(svc.Subsets[0].Addresses) == 0 {
		log.Printf("no addresses in subset for \"%s.%s\"", functionName, namespace)
	}

	target := rand.Intn(all)

	serviceIP := svc.Subsets[0].Addresses[target].IP

	urlStr := fmt.Sprintf("http://%s:%d", serviceIP, watchdogPort)

	// urlRes, err := url.Parse(urlStr)
	// if err != nil {
	// 	return url.URL{}, err
	// }

	log.Printf("[Call k8s/proxy.go Resolve] name: %s, url %s", name, urlStr)

	urlresp, err := http.Get(urlStr)
	//defer resp.Body.Close()
	if err != nil {
		log.Printf(err.Error())
	}
	// body, err := ioutil.ReadAll(resp.Body)
	// fmt.Printf("response is :%s \n", string(body))

	return urlresp, err
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
