package model

type RegisterOpenIdData struct {
	OpenId   string `form:"openId" binding:"required" validate:"max=32, min=32"`
	Value    string `form:"value" binding:"required"`
	Location string `form:"location"` // 新增字段：用于接收经纬度字符串，格式为 "经度,纬度"
}
