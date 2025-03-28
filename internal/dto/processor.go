package dto

import "github.com/chekist32/goipay/internal/db"

type NewInvoiceRequest struct {
	UserId        string
	Coin          db.CoinType
	Amount        float64
	Timeout       uint64
	Confirmations uint32
}

type DaemonConfig struct {
	Url  string
	User string
	Pass string
}
type XMRDaemonConfig DaemonConfig
type BTCDaemonConfig DaemonConfig
type LTCDaemonConfig DaemonConfig
type ETHDaemonConfig DaemonConfig
type BNBDaemonConfig ETHDaemonConfig

type DaemonsConfig struct {
	Xmr XMRDaemonConfig
	Btc BTCDaemonConfig
	Ltc LTCDaemonConfig
	Eth ETHDaemonConfig
	Bnb BNBDaemonConfig
}
