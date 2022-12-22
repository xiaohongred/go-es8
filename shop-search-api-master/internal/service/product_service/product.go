package product_service

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

type Product struct {
	UserID     int64  `json:"userid" bson:"userid"`
	Keyword    string `json:"keyword" bson:"keyword"`
	New        *int   `json:"new" bson:"new"`
	Sales      string `json:"sales" bson:"sales"`
	Price      string `json:"price" bson:"price"`
	PageNum    int    `json:"page_num" bson:"page_num"`
	PageSize   int    `json:"page_size" bson:"page_size"`
	CreateTime int64  `json:"create_time" bson:"create_time"`
}

func (p *Product) SearchProduct() (result *elastic.SearchResult, err error) {
	query := elastic.NewBoolQuery()
	from := p.PageNum * 20

	query.MinimumNumberShouldMatch(1)

	storeNameMatchPhreaseQuery := elastic.NewMatchPhraseQuery("store_name", p.Keyword).Boost(2).QueryName("storeNameMatchPhreaseQuery")
	storeNameMatchQuery := elastic.NewMatchPhraseQuery("store_name", p.Keyword).Boost(1).QueryName("storeNameMatchQuery")
	storeNamePinyinMatchPhreaseQuery := elastic.NewMatchPhraseQuery("store_name.pinyin", p.Keyword).Boost(0.7).QueryName("storeNamePinyinMatchPhreaseQuery")
	descMatchQuery := elastic.NewMatchPhraseQuery("desc", p.Keyword).Boost(0.5).QueryName("descMatchQuery")

	shouldQuerys := make([]elastic.Query, 0)
	shouldQuerys = append(shouldQuerys, storeNameMatchPhreaseQuery, storeNameMatchQuery, descMatchQuery)

	if strutil.IncludeLetter(p.Keyword) {
		shouldQuerys = append(shouldQuerys, storeNamePinyinMatchPhreaseQuery)
	}

	//是否新品
	if p.New != nil {
		query.Must(elastic.NewTermQuery("is_new", p.New))
	}

	query.Should(shouldQuerys...)

	orders := make([]map[string]bool, 0)
	//价格排序
	if len(p.Price) > 0 {
		if strings.ToLower(p.Price) == "desc" {
			orders = append(orders, map[string]bool{"price": false})
		} else {
			orders = append(orders, map[string]bool{"price": true})
		}
	}

	//销量排序
	if len(p.Sales) > 0 {
		if strings.ToLower(p.Sales) == "desc" {
			orders = append(orders, map[string]bool{"sales": false})
		} else {
			orders = append(orders, map[string]bool{"sales": true})
		}
	}
	//默认按照相关度算分来排序
	orders = append(orders, map[string]bool{"_score": false})

	return global.ES.Query(context.Background(), global.ProductIndexName,
		nil, query, from, p.PageSize, es.WithEnableDSL(true),
		es.WithPreference(strutil.Int64ToString(p.UserID)),
		es.WithFetchSource(false), es.WithOrders(orders))
}

func (p *Product) LogReport() {
	if global.Mongo == nil {
		global.LOG.Error(" global.Mongo is nil", zap.Any("param", p))
		return
	}
	tablename := fmt.Sprintf(global.ProductSearchLogCollectionNamePrefix, timeutil.YMDLayoutInt64(time.Now()))
	//本地缓存，避免每次写入都要创建索引
	if _, ok := LogTableCreated.Load(tablename); !ok {
		err := global.Mongo.CreateMultiIndex(global.SearchLogDbName, tablename, []string{"userid", "create_time"}, false)
		if err != nil {
			global.LOG.Error(" Mongo CreateMultiIndex error", zap.Error(err), zap.Any("param", p))
		}
		LogTableCreated.Store(tablename, true)
	}
	err := global.Mongo.InsertMany(global.SearchLogDbName, tablename, p)
	if err != nil {
		global.LOG.Error(" Mongo InsertMany error", zap.Error(err), zap.Any("param", p))
	}
}
