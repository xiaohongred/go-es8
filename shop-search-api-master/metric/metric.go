package metric

import (
	"gitee.com/phper95/pkg/prome"
	"github.com/prometheus/client_golang/prometheus"
	"shop-search-api/config"
)

var ProductSearch = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:        "shop_product_search",
		Help:        "histogram for product search",
		Buckets:     prome.DefaultBuckets,
		ConstLabels: prometheus.Labels{"machine": prome.GetHostName(), "app": config.AppName},
	},
	[]string{"cluster"},
)

var OrderSearch = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:        "shop_order_search",
		Help:        "histogram for order search",
		Buckets:     prome.DefaultBuckets,
		ConstLabels: prometheus.Labels{"machine": prome.GetHostName(), "app": config.AppName},
	},
	[]string{"cluster"},
)
