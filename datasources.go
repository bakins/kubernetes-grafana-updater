package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

var datasourcesCmd = &cobra.Command{
	Use:   "datasources",
	Short: "syncronize datasources",
	Run:   runDatasourcesSync,
}

func init() {
	addK8sflags(datasourcesCmd)
	addGrafanaFlags(datasourcesCmd)

	datasourcesCmd.PersistentFlags().StringVarP(&selector, "selector", "", "app=prometheus", "label selector")
	rootCmd.AddCommand(datasourcesCmd)
}

type serviceSyncer struct {
	grafana *datasourcesClient
	k8s     *kubernetes.Clientset
}

func runDatasourcesSync(cmd *cobra.Command, args []string) {
	s := &serviceSyncer{
		grafana: newDatasourcesClient(getGrafanaURL(), nil),
		k8s:     newK8sClient(),
	}

	watchlist := newListWatchFromClient(
		s.k8s.CoreV1().RESTClient(),
		"services",
		namespace,
		selector,
	)

	_, controller := cache.NewInformer(
		watchlist,
		&v1.Service{},
		time.Second*60, // TODO: flag for this
		s,
	)

	stopChan := make(chan struct{})
	go controller.Run(stopChan)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	stopChan <- struct{}{}
}

// just return the first port for now.
// TODO: look for a specific name?
func getServicePort(s *v1.Service) int32 {
	if len(s.Spec.Ports) == 0 {
		return 9200
	}
	return s.Spec.Ports[0].Port
}

func getServiceName(s *v1.Service) string {
	return s.GetName() + "." + s.GetNamespace()
}

func (s *serviceSyncer) OnAdd(obj interface{}) {
	service, ok := obj.(*v1.Service)
	if !ok {
		return
	}

	name := getServiceName(service)

	// check if already exists
	existing, err := s.grafana.Get(name)
	if err != nil {
		log.Println(err)
		return
	}

	d := &grafanaDatasource{
		Name:   name,
		Type:   "prometheus",
		Access: "proxy",
		URL:    fmt.Sprintf("http://%s:%d", name, getServicePort(service)),
	}

	if existing == nil {
		err = s.grafana.Create(d)
		if err != nil {
			log.Println(err)
			return
		}
		return
	}

	d.ID = existing.ID
	err = s.grafana.Update(d)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func (s *serviceSyncer) OnDelete(obj interface{}) {
	service, ok := obj.(*v1.Service)
	if !ok {
		return
	}
	name := getServiceName(service)
	// check if already exists
	existing, err := s.grafana.Get(name)
	if err != nil {
		// TODO: log
		return
	}

	if existing == nil {
		// nothing to do
		return
	}

	err = s.grafana.Delete(existing.ID)
	if err != nil {
		// TODO: log
		return
	}
	return
}

func (s *serviceSyncer) OnUpdate(oldObj, newObj interface{}) {
	s.OnAdd(newObj)
}
