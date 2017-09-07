package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

var dashboardsCmd = &cobra.Command{
	Use:   "dashboards",
	Short: "syncronize dashboards",
	Run:   runDashboardsSync,
}

var dashboardSelector = "type=grafana-dashboard"

func init() {
	addK8sflags(dashboardsCmd)
	addGrafanaFlags(dashboardsCmd)

	dashboardsCmd.PersistentFlags().StringVarP(&dashboardSelector, "selector", "", "type=grafana-dashboard", "label selector")
	rootCmd.AddCommand(dashboardsCmd)
}

type configmapSyncer struct {
	grafana *grafanaClient
	k8s     *kubernetes.Clientset
}

func runDashboardsSync(cmd *cobra.Command, args []string) {
	s := &configmapSyncer{
		grafana: newGrafanaClient(getGrafanaURL(), nil),
		k8s:     newK8sClient(),
	}

	s.grafana.wait()

	watchlist := newListWatchFromClient(
		s.k8s.CoreV1().RESTClient(),
		"configmaps",
		namespace,
		dashboardSelector,
	)

	_, controller := cache.NewInformer(
		watchlist,
		&v1.ConfigMap{},
		time.Second*300, // TODO: flag for this
		s,
	)

	logger.Info("successfully started")

	stopChan := make(chan struct{})
	go controller.Run(stopChan)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	stopChan <- struct{}{}
}

// we should somehow get the slug as that is how it
// referenced inside grafana
func getDashboardName(c *v1.ConfigMap, filename string) string {
	extension := filepath.Ext(filename)
	name := filename[0 : len(filename)-len(extension)]

	return strings.ToLower(filepath.Base(name) + "-" + c.GetName() + "-" + c.GetNamespace())
}

func getDashboardTitle(c *v1.ConfigMap, filename string) string {
	extension := filepath.Ext(filename)
	name := filename[0 : len(filename)-len(extension)]

	return filepath.Base(name) + " " + c.GetName() + " " + c.GetNamespace()
}

func (s *configmapSyncer) OnAdd(obj interface{}) {
	configmap, ok := obj.(*v1.ConfigMap)
	if !ok {
		return
	}

	logger.Debug("got configmap", zap.String("name", configmap.ObjectMeta.Name))

	for k, v := range configmap.Data {
		name := getDashboardName(configmap, k)

		logger.Debug("getting dashboard", zap.String("name", name))

		// check if already exists
		existing, err := s.grafana.GetDashboard(name)
		if err != nil {
			log.Println(err)
			continue
		}

		var dashboard grafanaDashboard

		if err := json.Unmarshal([]byte(v), &dashboard.Model); err != nil {
			logger.Error("failed to unmarshal dashboard", zap.String("name", name), zap.Error(err))
			continue
		}

		dashboard.Model["title"] = name

		if existing == nil {
			logger.Debug("creating dashboard", zap.String("name", name))
			err = s.grafana.CreateDashboard(&dashboard)
			if err != nil {
				log.Println(err)

			}
			continue
		}

		logger.Debug("updating dashboard", zap.String("name", name))
		err = s.grafana.UpdateDashboard(&dashboard)
		if err != nil {
			log.Println(err)
		}
	}

}

func (s *configmapSyncer) OnDelete(obj interface{}) {
	// do nothing. to delete, just restart grafana
	// and this will add dashboards back
}

func (s *configmapSyncer) OnUpdate(oldObj, newObj interface{}) {
	s.OnAdd(newObj)
}
