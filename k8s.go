package main

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig string
	apiserver  string
	namespace  string
	selector   string
)

func addK8sflags(cmd *cobra.Command) {
	f := cmd.PersistentFlags()
	f.StringVarP(&apiserver, "apiserver", "", "", "override Kubernetes API server. default is to use value from kubeconfig or in cluster value")
	f.StringVarP(&kubeconfig, "kubeconfig", "", "", "path to kubeconfig. default is in cluster.")
	f.StringVarP(&namespace, "namespace", "", "", "namespace to search. Default is all namespaces")
}

func newK8sClient() *kubernetes.Clientset {

	config, err := clientcmd.BuildConfigFromFlags(apiserver, kubeconfig)
	if err != nil {
		errorLog("failed to build kubernetes config: %s", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		errorLog("failed to create kubernetes client: %s", err)
	}

	return clientset
}

// based on implementation in k8s. modified to use label selector
func newListWatchFromClient(c cache.Getter, resource, namespace, selector string) *cache.ListWatch {
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = selector
		logger.Info("listFunc", zap.String("selector", selector))
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Do().
			Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.LabelSelector = selector
		logger.Info("watchFunc", zap.String("selector", selector))
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec).
			Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func newWatchList(k8s *kubernetes.Clientset, objType, namespace, selector string) *cache.ListWatch {
	watchlist := cache.NewListWatchFromClient(k8s.CoreV1().RESTClient(), objType, namespace, fields.Everything())

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = selector
		return watchlist.ListFunc(options)
	}
	watchlist.ListFunc = listFunc

	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.LabelSelector = selector
		return watchlist.WatchFunc(options)
	}
	watchlist.WatchFunc = watchFunc

	return watchlist

}
