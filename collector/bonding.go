// +build !nobonding

package collector

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	sysfsNet = "/sys/class/net"
)

var (
	bondingSlaves = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "net_bonding_slaves",
			Help:      "Number of configured slaves per bonding interface.",
		}, []string{"master"})
	bondingSlavesActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "net_bonding_slaves_active",
			Help:      "Number of active slaves per bonding interface.",
		}, []string{"master"})
)

type bondingCollector struct{}

func init() {
	Factories["bonding"] = NewBondingCollector
}

// NewBondingCollector returns a newly allocated bondingCollector.
// It exposes the number of configured and active slave of linux bonding interfaces.
func NewBondingCollector(config Config) (Collector, error) {
	c := bondingCollector{}
	return &c, nil
}

// Update reads and exposes bonding states, implements Collector interface. Caution: This works only on linux.
func (c *bondingCollector) Update(ch chan<- prometheus.Metric) (err error) {
	bondingStats, err := readBondingStats(sysfsNet)
	if err != nil {
		return err
	}
	for master, status := range bondingStats {
		bondingSlaves.WithLabelValues(master).Set(float64(status[0]))
		bondingSlavesActive.WithLabelValues(master).Set(float64(status[1]))
	}
	bondingSlaves.Collect(ch)
	bondingSlavesActive.Collect(ch)
	return nil
}

func readBondingStats(root string) (status map[string][2]int, err error) {
	status = map[string][2]int{}
	masters, err := ioutil.ReadFile(path.Join(root, "bonding_masters"))
	if err != nil {
		return nil, err
	}
	for _, master := range strings.Fields(string(masters)) {
		slaves, err := ioutil.ReadFile(path.Join(root, master, "bonding", "slaves"))
		if err != nil {
			return nil, err
		}
		sstat := [2]int{0, 0}
		for _, slave := range strings.Fields(string(slaves)) {
			state, err := ioutil.ReadFile(path.Join(root, master, fmt.Sprintf("lower_%s", slave), "operstate"))
			if os.IsNotExist(err) {
				// some older? kernels use slave_ prefix
				state, err = ioutil.ReadFile(path.Join(root, master, fmt.Sprintf("slave_%s", slave), "operstate"))
			}
			if err != nil {
				return nil, err
			}
			sstat[0]++
			if strings.TrimSpace(string(state)) == "up" {
				sstat[1]++
			}
		}
		status[master] = sstat
	}
	return status, err
}
