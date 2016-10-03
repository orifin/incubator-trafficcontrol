package deliveryservicedata // TODO rename?

import (
	"errors"
	"github.com/Comcast/traffic_control/traffic_monitor/experimental/traffic_monitor/enum"
)

// Filter encapsulates functions to filter a given set of Stats, e.g. from HTTP query parameters.
// TODO combine with cache.Filter?
type Filter interface {
	UseStat(name string) bool
	UseDeliveryService(name enum.DeliveryServiceName) bool
	WithinStatHistoryMax(int) bool
}

type StatName string
type StatOld struct {
	Time  int64  `json:"time"`
	Value string `json:"value"`
	Span  int    `json:"span,omitempty"`  // TODO set? remove?
	Index int    `json:"index,omitempty"` // TODO set? remove?
}
type StatsOld struct {
	DeliveryService map[enum.DeliveryServiceName]map[StatName][]StatOld `json:"deliveryService"`
}

type StatsReadonly interface {
	Get(name enum.DeliveryServiceName) (StatReadonly, bool)
	JSON(filter Filter) StatsOld
}

type StatReadonly interface {
	Copy() Stat
	Common() StatCommonReadonly
	CacheGroup(name enum.CacheGroupName) (StatCacheStats, bool)
	Type(name enum.CacheType) (StatCacheStats, bool)
	Total() StatCacheStats
}

type StatCommonReadonly interface {
	Copy() StatCommon
	CachesConfigured() StatInt
	CachesReportingNames() []enum.CacheName
	Error() StatString
	Status() StatString
	Healthy() StatBool
	Available() StatBool
	CachesAvailable() StatInt
}

// New, more structured format:
type StatMeta struct {
	Time int `json:"time"`
}
type StatFloat struct {
	StatMeta
	Value float64 `json:"value"`
}
type StatBool struct {
	StatMeta
	Value bool `json:"value"`
}
type StatInt struct {
	StatMeta
	Value int64 `json:"value"`
}
type StatString struct {
	StatMeta
	Value string `json:"value"`
}

type StatCommon struct {
	CachesConfiguredNum StatInt                 `json:"caches_configured"`
	CachesReporting     map[enum.CacheName]bool `json:"caches_reporting"`
	ErrorStr            StatString              `json:"error_string"`
	StatusStr           StatString              `json:"status"`
	IsHealthy           StatBool                `json:"is_healthy"`
	IsAvailable         StatBool                `json:"is_available"`
	CachesAvailableNum  StatInt                 `json:"caches_available"`
}

func (a StatCommon) Copy() StatCommon {
	b := a
	for k, v := range a.CachesReporting {
		b.CachesReporting[k] = v
	}
	return b
}

func (a StatCommon) CachesConfigured() StatInt {
	return a.CachesConfiguredNum
}
func (a StatCommon) CacheReporting(name enum.CacheName) (bool, bool) {
	c, ok := a.CachesReporting[name]
	return c, ok
}
func (a StatCommon) CachesReportingNames() []enum.CacheName {
	names := make([]enum.CacheName, 0, len(a.CachesReporting))
	for name, _ := range a.CachesReporting {
		names = append(names, name)
	}
	return names
}
func (a StatCommon) Error() StatString {
	return a.ErrorStr
}
func (a StatCommon) Status() StatString {
	return a.StatusStr
}
func (a StatCommon) Healthy() StatBool {
	return a.IsHealthy
}
func (a StatCommon) Available() StatBool {
	return a.IsAvailable
}
func (a StatCommon) CachesAvailable() StatInt {
	return a.CachesAvailableNum
}

// StatCacheStats is all the stats generated by a cache.
// This may also be used for aggregate stats, for example, the summary of all cache stats for a cache group, or delivery service.
// Each stat is an array, in case there are multiple data points at different times. However, a single data point i.e. a single array member is common.
type StatCacheStats struct {
	OutBytes    StatInt    `json:"out_bytes"`
	IsAvailable StatBool   `json:"is_available"`
	Status5xx   StatInt    `json:"status_5xx"`
	Status4xx   StatInt    `json:"status_4xx"`
	Status3xx   StatInt    `json:"status_3xx"`
	Status2xx   StatInt    `json:"status_2xx"`
	InBytes     StatFloat  `json:"in_bytes"`
	Kbps        StatFloat  `json:"kbps"`
	Tps5xx      StatInt    `json:"tps_5xx"`
	Tps4xx      StatInt    `json:"tps_4xx"`
	Tps3xx      StatInt    `json:"tps_3xx"`
	Tps2xx      StatInt    `json:"tps_2xx"`
	ErrorString StatString `json:"error_string"`
	TpsTotal    StatInt    `json:"tps_total"`
}

func (a StatCacheStats) Sum(b StatCacheStats) StatCacheStats {
	return StatCacheStats{
		OutBytes:    StatInt{Value: a.OutBytes.Value + b.OutBytes.Value},
		IsAvailable: StatBool{Value: a.IsAvailable.Value || b.IsAvailable.Value},
		Status5xx:   StatInt{Value: a.Status5xx.Value + b.Status5xx.Value},
		Status4xx:   StatInt{Value: a.Status4xx.Value + b.Status4xx.Value},
		Status3xx:   StatInt{Value: a.Status3xx.Value + b.Status3xx.Value},
		Status2xx:   StatInt{Value: a.Status2xx.Value + b.Status2xx.Value},
		InBytes:     StatFloat{Value: a.InBytes.Value + b.InBytes.Value},
		Kbps:        StatFloat{Value: a.Kbps.Value + b.Kbps.Value},
		Tps5xx:      StatInt{Value: a.Tps5xx.Value + b.Tps5xx.Value},
		Tps4xx:      StatInt{Value: a.Tps4xx.Value + b.Tps4xx.Value},
		Tps3xx:      StatInt{Value: a.Tps3xx.Value + b.Tps3xx.Value},
		Tps2xx:      StatInt{Value: a.Tps2xx.Value + b.Tps2xx.Value},
		ErrorString: StatString{Value: a.ErrorString.Value + b.ErrorString.Value},
		TpsTotal:    StatInt{Value: a.TpsTotal.Value + b.TpsTotal.Value},
	}
}

type Stat struct {
	CommonStats StatCommon
	CacheGroups map[enum.CacheGroupName]StatCacheStats
	Types       map[enum.CacheType]StatCacheStats
	TotalStats  StatCacheStats
}

var ErrNotProcessedStat = errors.New("This stat is not used.")

func NewStat() *Stat {
	return &Stat{CacheGroups: map[enum.CacheGroupName]StatCacheStats{}, Types: map[enum.CacheType]StatCacheStats{}, CommonStats: StatCommon{CachesReporting: map[enum.CacheName]bool{}}}
}

func (a Stat) Copy() Stat {
	b := Stat{CommonStats: a.CommonStats.Copy(), TotalStats: a.TotalStats, CacheGroups: map[enum.CacheGroupName]StatCacheStats{}, Types: map[enum.CacheType]StatCacheStats{}}
	for k, v := range a.CacheGroups {
		b.CacheGroups[k] = v
	}
	for k, v := range a.Types {
		b.Types[k] = v
	}
	return b
}

func (a Stat) Common() StatCommonReadonly {
	return a.CommonStats
}

func (a Stat) CacheGroup(name enum.CacheGroupName) (StatCacheStats, bool) {
	c, ok := a.CacheGroups[name]
	return c, ok
}

func (a Stat) Type(name enum.CacheType) (StatCacheStats, bool) {
	t, ok := a.Types[name]
	return t, ok
}

func (a Stat) Total() StatCacheStats {
	return a.TotalStats
}
