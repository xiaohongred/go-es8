package order_service

import (
	"context"
	"fmt"
	"gitee.com/phper95/pkg/es"
	"gitee.com/phper95/pkg/strutil"
	"gitee.com/phper95/pkg/timeutil"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"shop-search-api/global"
	"strings"
	"sync"
	"time"
)

var (
	LogTableCreated sync.Map
)

type Order struct {
	UserID         int64  `json:"userid" bson:"userid"`
	Keyword        string `json:"keyword" bson:"keyword"`
	PageNum        int    `json:"page_num" bson:"page_num"`
	PageSize       int    `json:"page_size" bson:"page_size"`
	OrderStatus    *int   `json:"order_status" bson:"order_status"`
	CreateTimeSort string `json:"create_time_sort" bson:"create_time_sort"`
	UpdateTimeSort string `json:"update_time_sort" bson:"update_time_sort"`
	LogTime        int64  `json:"log_time" bson:"log_time"`
}

func (o *Order) SearchOrder() (result *elastic.SearchResult, err error) {
	query := elastic.NewBoolQuery()
	from := o.PageNum * 20

	query.MinimumNumberShouldMatch(1)

	namesMatchPhreaseQuery := elastic.NewMatchPhraseQuery("names", o.Keyword).Boost(2).QueryName("namesMatchPhreaseQuery")
	namesMatchQuery := elastic.NewMatchQuery("names", o.Keyword).Boost(1).QueryName("namesMatchQuery")
	namesPinyinMatchPhreaseQuery := elastic.NewMatchPhraseQuery("names.pinyin", o.Keyword).Boost(0.7).QueryName("namesPinyinMatchPhreaseQuery")
	orderIDMatchPhraseQuery := elastic.NewMatchPhraseQuery("order_id", o.Keyword).Boost(0.5).QueryName("orderIDMatchPhraseQuery")
	orderIDSuffixMatchPhraseQuery := elastic.NewMatchPhraseQuery("order_id_suffix", o.Keyword).Boost(0.3).QueryName("orderIDSuffixMatchPhraseQuery")

	shouldQuerys := make([]elastic.Query, 0)
	shouldQuerys = append(shouldQuerys, namesMatchPhreaseQuery, namesMatchQuery)

	mustQuerys := make([]elastic.Query, 0)
	uidMustQuery := elastic.NewTermQuery("uid", o.UserID)
	orderStatusMustQuery := elastic.NewTermQuery("order_status", o.OrderStatus)
	mustQuerys = append(mustQuerys, uidMustQuery)

	if o.OrderStatus != nil {
		mustQuerys = append(mustQuerys, orderStatusMustQuery)
	}

	//高亮字段
	highlight := elastic.NewHighlight()
	highlight.NumOfFragments(1) //默认值5
	highlight.FragmentSize(100) //默认值100
	highlight.Field("names")

	if strutil.IncludeLetter(o.Keyword) {
		shouldQuerys = append(shouldQuerys, namesPinyinMatchPhreaseQuery)
		highlight.Field("names.pinyin")
	}

	//尽可能减少不必要的查询条件
	if strutil.IsDigit(o.Keyword) {
		shouldQuerys = append(shouldQuerys, orderIDMatchPhraseQuery, orderIDSuffixMatchPhraseQuery)
		highlight.Field("order_id")
		highlight.Field("order_id_suffix")
	}

	//过滤当前用户的订单
	query.Must(mustQuerys...)

	query.Should(shouldQuerys...)

	orders := make([]map[string]bool, 0)
	//更新时间排序
	if len(o.UpdateTimeSort) > 0 {
		if strings.ToLower(o.UpdateTimeSort) == "desc" {
			orders = append(orders, map[string]bool{"update_time": false})
		} else {
			orders = append(orders, map[string]bool{"update_time": true})
		}
	}

	//创建时间排序
	if len(o.CreateTimeSort) > 0 {
		if strings.ToLower(o.CreateTimeSort) == "desc" {
			orders = append(orders, map[string]bool{"create_time": false})
		} else {
			orders = append(orders, map[string]bool{"create_time": true})
		}
	}
	//默认按照相关度算分来排序
	orders = append(orders, map[string]bool{"_score": false})

	//注意，查询的时候使用UserID作routing
	return global.ES.Query(context.Background(), global.OrderIndexName,
		[]string{strutil.Int64ToString(o.UserID)}, query, from, o.PageSize, es.WithEnableDSL(true),
		es.WithPreference(strutil.Int64ToString(o.UserID)),
		es.WithFetchSource(false), es.WithOrders(orders),
		es.WithHighlight(highlight))
}

func (o *Order) LogReport() {
	if global.Mongo == nil {
		global.LOG.Error(" global.Mongo is nil", zap.Any("param", o))
		return
	}
	tablename := fmt.Sprintf(global.OderSearchLogCollectionNamePrefix, timeutil.YMDLayoutInt64(time.Now()))
	//本地缓存，避免每次写入都要创建索引
	if _, ok := LogTableCreated.Load(tablename); !ok {
		err := global.Mongo.CreateMultiIndex(global.SearchLogDbName, tablename, []string{"userid", "create_time"}, false)
		if err != nil {
			global.LOG.Error(" Mongo CreateMultiIndex error", zap.Error(err), zap.Any("param", o))
		}
		LogTableCreated.Store(tablename, true)
	}
	err := global.Mongo.InsertMany(global.SearchLogDbName, tablename, o)
	if err != nil {
		global.LOG.Error(" Mongo InsertMany error", zap.Error(err), zap.Any("param", o))
	}
}
