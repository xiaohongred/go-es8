package global

import (
	"gitee.com/phper95/pkg/cache"
	"gitee.com/phper95/pkg/es"
	"gitee.com/phper95/pkg/nosql"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ES    *es.Client
	LOG   *zap.Logger
	DB    *gorm.DB
	CACHE *cache.Redis
	Mongo *nosql.MgClient
)
