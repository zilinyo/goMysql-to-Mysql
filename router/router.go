package router

import (
	"cardappcanal/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

func SetServiceMiddleware(service *service.TransferService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("transfer-service", service)
		c.Next()
	}
}
func InitRouter(service *service.TransferService) *gin.Engine {
	router := gin.Default()
	router.Use(SetServiceMiddleware(service))
	router.GET("/info", getInfoHandler)
	router.GET("/performAction", performActionHandler)
	return router
}

func getInfoHandler(c *gin.Context) {
	// 处理获取信息的请求逻辑
	// 构建 Response 结构体
	serviceValue, exists := c.Get("transfer-service")
	if !exists {
		c.String(http.StatusInternalServerError, "Service not found in context")
		return
	}

	// Check if the service has the expected type
	service, ok := serviceValue.(*service.TransferService)
	if !ok {
		c.String(http.StatusInternalServerError, "Unexpected type for service in context")
		return
	}
	service.Close()
	// 返回 JSON 数据
	c.JSON(http.StatusOK, "")
}

func performActionHandler(c *gin.Context) {

	// 返回 JSON 数据
	c.JSON(http.StatusOK, "")
}
