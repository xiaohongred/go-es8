package order_repo

import "time"

type OrderIndex struct {
	OrderId       string    `json:"order_id"`
	OrderIdSuffix string    `json:"order_id_suffix"`
	Names         []string  `json:"names"`
	ProductIds    []int64   `json:"product_ids"`
	Uid           int64     `json:"uid"`
	PayTime       time.Time `json:"pay_time"`
	PayType       string    `json:"pay_type"`
	RefundStatus  int       `json:"refund_status"` // '0 未退款 1 申请中 2 已退款'
	ShippingType  int       `json:"shipping_type"` //配送方式
	OrderStatus   int       `json:"order_status"`
	CreateTime    time.Time `json:"create_time"`
	UpdateTime    time.Time `json:"update_time"`
}
