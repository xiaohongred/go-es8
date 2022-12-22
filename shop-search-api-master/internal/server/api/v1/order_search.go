package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/unknwon/com"
	"go.uber.org/zap"
	"shop-search-api/global"
	"shop-search-api/internal/pkg/errcode"
	"shop-search-api/internal/repo/es/order_repo"
	"shop-search-api/internal/server/api/api_response"
	"shop-search-api/internal/service/order_service"
	"shop-search-api/metric"
	"time"
)

type orderSerchResponse struct {
	Total int64          `json:"total"`
	Hits  []*orderResult `json:"hits"`
}

type orderResult struct {
	order_repo.OrderIndex
	Highlight map[string][]string `json:"highlight"`
}

func OrderSearch(c *gin.Context) {
	t := time.Now()
	cluster := "a"
	//监控上报
	defer func() {
		obs, err := metric.OrderSearch.GetMetricWithLabelValues(cluster)
		if err != nil {
			global.LOG.Error("metric.OrderSearch error", zap.Error(err))
		} else {
			obs.Observe(float64(time.Since(t).Milliseconds()))
		}
	}()
	appG := api_response.Gin{C: c}
	keyword := c.Query("keyword")
	if len(keyword) == 0 {
		appG.ResponseErr(errcode.ErrCodes.ErrParams)
		return
	}
	orderService := order_service.Order{
		Keyword:        keyword,
		PageNum:        com.StrTo(c.Query("page_num")).MustInt(),
		PageSize:       com.StrTo(c.Query("page_size")).MustInt(),
		UserID:         com.StrTo(c.Query("userid")).MustInt64(),
		UpdateTimeSort: c.Query("update_time_sort"),
		CreateTimeSort: c.Query("create_time_sort"),
	}
	orderStatus := -1
	orderStatusStr := c.Query("order_status")
	if len(orderStatusStr) > 0 {
		if orderStatus = com.StrTo(orderStatusStr).MustInt(); orderStatus > 0 {
			orderService.OrderStatus = &orderStatus
		}
	}

	//上报搜索日志
	orderService.LogTime = time.Now().Unix()
	defer func() {
		orderService.LogReport()
	}()

	//模拟多集群上报
	if orderService.UserID%2 == 0 {
		cluster = "b"
	}

	res, err := orderService.SearchOrder()
	global.LOG.Warn("resp", zap.Any("", res))
	if err != nil {
		global.LOG.Error("search error", zap.Error(err), zap.Any("param", orderService))
		appG.ResponseErr(errcode.ErrCodes.ErrSearch)
		return
	}
	resp := orderSerchResponse{
		Total: 0,
		Hits:  make([]*orderResult, 0),
	}
	if res == nil {
		appG.ResponseOk(errcode.ErrCodes.ErrNo, resp)
		return
	}
	resp.Total = res.Hits.TotalHits.Value
	for _, hit := range res.Hits.Hits {
		index := &orderResult{}
		//err = json.Unmarshal(hit.Source, index)
		//if err != nil {
		//	global.LOG.Error("Unmarshal error", zap.Error(err))
		//	continue
		//}
		index.OrderId = hit.Id
		index.Highlight = hit.Highlight
		resp.Hits = append(resp.Hits, index)
	}
	global.LOG.Warn("resp", zap.Any("resp", resp))
	appG.ResponseOk(errcode.ErrCodes.ErrNo, resp)
}
