package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Popilot struct {
	ClientID    string
	DeviceCode  string
	UserCode    string
	AccessToken string
	TID         string
	Exp         int64
	TLT         string
}

var popilotDB = []Popilot{}

// coauth 函数用于配置和初始化 Gin 引擎的路由。
// 它读取配置文件，并设置多个 HTTP 路由处理函数，以实现设备登录、OAuth 认证和用户信息获取等功能。
//
// 参数:
//   - r: 指向 gin.Engine 的指针，用于注册路由和处理函数。
//
// 路由说明:
//   - GET "/"：返回欢迎信息 "Hello Popilot!"。
//   - GET "/login/device"：返回提示信息 "请关闭此页面"。
//   - POST "/login/device/code"：生成设备代码和用户代码，并返回相关信息。
//   - POST "/login/oauth/access_token"：根据设备代码和客户端ID生成访问令牌。
//   - GET "/api/v3/user"：返回模拟的用户信息。
//   - GET "/api/v3/meta"：返回空的元信息。
//   - GET "/copilot_internal/v2/token"：根据访问令牌生成跟踪ID和过期时间，并返回相关配置信息。
//   - GET "/teams/:team/memberships/:membership"：返回404错误，提示未找到。
//
// 注意:
//   - 该函数依赖于外部配置读取函数 readConfig 和全局变量 popilotDB。
//   - 使用了 uuid 包生成唯一标识符。
//   - 使用了 gin 框架进行 HTTP 请求处理。
func coauth(r *gin.Engine) {
	cfg := readConfig()
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello Popilot!")
	})
	r.GET("/login/device", func(c *gin.Context) {
		c.String(http.StatusOK, "请关闭此页面")
	})

	r.POST("/login/device/code", func(c *gin.Context) {
		clientID := c.Query("client_id")
		deviceCode := uuid.New().String()
		userCode := strings.ToUpper(uuid.New().String()[:4]) + "-" + strings.ToUpper(uuid.New().String()[:4])

		// 检查是否存在相同的 client_id 并删除
		for i, item := range popilotDB {
			if item.ClientID == clientID {
				popilotDB = append(popilotDB[:i], popilotDB[i+1:]...)
			}
		}

		// 插入新的设备代码
		popilotDB = append(popilotDB, Popilot{
			ClientID:   clientID,
			DeviceCode: deviceCode,
			UserCode:   userCode,
		})
		port := strings.Split(cfg.Bind, ":")[1]
		res := map[string]interface{}{
			"device_code":      deviceCode,
			"expires_in":       900,
			"interval":         5,
			"user_code":        userCode,
			"verification_uri": "http://localhost:" + port + "/login/device",
		}
		c.JSON(http.StatusOK, res)
	})

	r.POST("/login/oauth/access_token", func(c *gin.Context) {
		deviceCode := c.Query("device_code")
		clientID := c.Query("client_id")

		var matched *Popilot
		for i, item := range popilotDB {
			if item.DeviceCode == deviceCode && item.ClientID == clientID {
				matched = &popilotDB[i]
				break
			}
		}

		if matched != nil {
			accessToken := "ccu_" + uuid.New().String()
			matched.AccessToken = accessToken

			res := map[string]interface{}{
				"access_token": accessToken,
				"scope":        "user:email",
				"token_type":   "bearer",
			}
			c.JSON(http.StatusOK, res)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		}
	})

	r.GET("/api/v3/user", func(c *gin.Context) {
		res := map[string]interface{}{
			"avatar_url": "https://avatars.githubusercontent.com/u/0?v=4",
			"id":         114514,
			"lid":        114514,
			"login":      "野兽先辈",
			"name":       "野兽先辈",
			"site_admin": false,
			"type":       "User",
		}
		c.JSON(http.StatusOK, res)
	})

	r.GET("/api/v3/meta", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{})
	})

	r.GET("/copilot_internal/v2/token", func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if len(token) > 0 {
			token = strings.Split(token, " ")[1]
		}

		var matched *Popilot
		for i, item := range popilotDB {
			if item.AccessToken == token {
				matched = &popilotDB[i]
				break
			}
		}

		if matched != nil {
			trackingID := uuid.New().String()
			exp := time.Now().Unix() + 3600
			tlt := uuid.New().String()

			matched.TID = trackingID
			matched.Exp = exp
			matched.TLT = tlt

			res := map[string]interface{}{
				"cocopilot_share_id":                       0,
				"annotations_enabled":                      false,
				"chat_enabled":                             true,
				"chat_jetbrains_enabled":                   true,
				"code_quote_enabled":                       true,
				"codesearch":                               false,
				"copilot_ide_agent_chat_gpt4_small_prompt": false,
				"copilotignore_enabled":                    false,
				"expires_at":                               exp,
				"individual":                               false,
				"intellij_editor_fetcher":                  false,
				"nes_enabled":                              false,
				"organization_list":                        nil,
				"prompt_8k":                                false,
				"public_suggestions":                       "disabled",
				"refresh_in":                               1500,
				"sku":                                      "yearly_subscriber",
				"snippy_load_test_enabled":                 false,
				"telemetry":                                "disabled",
				"token":                                    "tid=" + trackingID + ";exp=" + fmt.Sprint(exp) + ";sku=yearly_subscriber;st=dotcom;ssc=1;chat=1;8kp=0:" + tlt,
				"tracking_id":                              trackingID,
				"vsc_electron_fetcher":                     false,
				"vs_editor_fetcher":                        false,
				"vsc_panel_v2":                             false,
			}
			c.JSON(http.StatusOK, res)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		}
	})

	r.GET("/teams/:team/memberships/:membership", func(c *gin.Context) {
		// 你可以通过 c.Param("team") 和 c.Param("membership") 来获取路由参数
		c.JSON(http.StatusNotFound, gin.H{
			"documentation_url": "https://docs.github.com/rest",
			"message":           "Not Found",
		})
	})

}
