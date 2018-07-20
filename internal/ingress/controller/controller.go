/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"time"

	clientset "k8s.io/client-go/kubernetes"

	"github.com/kubernetes-sigs/aws-alb-ingress-controller/internal/albingress"
	"github.com/kubernetes-sigs/aws-alb-ingress-controller/internal/ingress/annotations/class"
)

// Configuration contains all the settings required by an Ingress controller
type Configuration struct {
	APIServerHost  string
	KubeConfigFile string
	Client         clientset.Interface

	ResyncPeriod time.Duration

	ConfigMapName string

	Namespace string

	DefaultHealthzURL     string
	DefaultSSLCertificate string

	ElectionID string

	HealthzPort int

	ClusterName             string
	ALBNamePrefix           string
	RestrictScheme          bool
	RestrictSchemeNamespace string
	AWSSyncPeriod           time.Duration
	AWSAPIMaxRetries        int
	AWSAPIDebug             bool

	EnableProfiling bool

	SyncRateLimit float32
}

// syncIngress collects all the pieces required to assemble the NGINX
// configuration file and passes the resulting data structures to the backend
// (OnUpdate) when a reload is deemed necessary.
func (c *ALBController) syncIngress(interface{}) error {
	c.syncRateLimiter.Accept()

	if c.syncQueue.IsShuttingDown() {
		return nil
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.metricCollector.IncReconcileCount()

	newIngresses := albingress.NewALBIngressesFromIngresses(&albingress.NewALBIngressesFromIngressesOptions{
		Recorder:            c.recorder,
		ClusterName:         c.cfg.ClusterName,
		ALBNamePrefix:       c.cfg.ALBNamePrefix,
		Store:               c.store,
		ALBIngresses:        c.runningConfig.Ingresses,
		IngressClass:        class.IngressClass,
		DefaultIngressClass: class.DefaultClass,
	})

	// // Update the prometheus gauge
	// ingressesByNamespace := map[string]int{}
	// logger.Debugf("Ingress count: %d", len(newIngresses))
	// for _, ingress := range newIngresses {
	// 	ingressesByNamespace[ingress.Namespace()]++
	// }

	// for ns, count := range ingressesByNamespace {
	// 	albprom.ManagedIngresses.With(
	// 		prometheus.Labels{"namespace": ns}).Set(float64(count))
	// }

	// Sync the state, resulting in creation, modify, delete, or no action, for every ALBIngress
	// instance known to the ALBIngress controller.
	removedIngresses := c.runningConfig.Ingresses.RemovedIngresses(newIngresses)

	// Update the list of ALBIngresses known to the ALBIngress controller to the newly generated list.
	c.runningConfig.Ingresses = newIngresses

	// // Reconcile the states
	removedIngresses.Reconcile()
	c.runningConfig.Ingresses.Reconcile()

	// err := c.OnUpdate(*pcfg)
	// if err != nil {
	// 	c.metricCollector.IncReloadErrorCount()
	// 	// c.metricCollector.ConfigSuccess(hash, false)
	// 	glog.Errorf("Unexpected failure reloading the backend:\n%v", err)
	// 	return err
	// }

	// c.metricCollector.ConfigSuccess(hash, true)
	// ri := getRemovedIngresses(c.runningConfig, pcfg)
	// re := getRemovedHosts(c.runningConfig, pcfg)
	// c.metricCollector.RemoveMetrics(ri, re)

	return nil
}
