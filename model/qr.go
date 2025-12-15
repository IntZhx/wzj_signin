package model

// 和WS服务建立连接通用结构，主要用于获得ClientID
type WSData struct {
	ID string `json:"id"`
	Channel string `json:"channel"`
	Successful bool `json:"successful"`
	Version string `json:"version"`
	SupportedConnectionTypes []string `json:"supportedConnectionTypes"`
	ClientID string `json:"clientId"`
	Advice Advice `json:"advice"`
}
type Advice struct {
	Reconnect string `json:"reconnect"`
	Interval int `json:"interval"`
	Timeout int `json:"timeout"`
}

// WS建立后，和二维码签到服务相关的结构，主要用于获得QrUrl
type QRCodeUrlData struct {
	Channel string `json:"channel"`
	Data Data `json:"data"`
	ID string `json:"id"`
	Ext Ext `json:"ext"`
}
type Data struct {
	Type int `json:"type"`
	QrURL string `json:"qrUrl"`
}
type Ext struct {
	InnerFayeToken string `json:"innerFayeToken"`
}
