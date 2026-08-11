package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
)

// AppServer 应用服务器结构体，封装所有服务和处理器
type AppServer struct {
	xiaohongshuService *XiaohongshuService
	mcpServer          *mcp.Server
	router             *gin.Engine
	httpServer         *http.Server
}

// NewAppServer 创建新的应用服务器实例
func NewAppServer(xiaohongshuService *XiaohongshuService) *AppServer {
	appServer := &AppServer{
		xiaohongshuService: xiaohongshuService,
	}

	// 初始化 MCP Server（需要在创建 appServer 之后，因为工具注册需要访问 appServer）
	appServer.mcpServer = InitMCPServer(appServer)

	return appServer
}

// Start 启动服务器
func (s *AppServer) Start(port string) error {
	s.router = setupRoutes(s)

	s.httpServer = &http.Server{
		Addr:    port,
		Handler: s.router,
	}

	// 启动服务器的 goroutine
	go func() {
		logrus.Infof("启动 HTTP 服务器: %s", port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("服务器启动失败: %v", err)
			os.Exit(1)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Infof("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		logrus.Warnf("等待连接关闭超时，强制退出: %v", err)
	} else {
		logrus.Infof("服务器已优雅关闭")
	}

	return nil
}

// corsMiddleware CORS 中间件，允许所有跨域请求
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 允许所有来源
		c.Header("Access-Control-Allow-Origin", "*")
		
		// 允许常见的请求方法
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		
		// 允许常见的请求头
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
		
		// 允许暴露的响应头
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		
		// 设置预检请求的缓存时间（可选）
		c.Header("Access-Control-Max-Age", "86400") // 24小时
		
		// 处理预检请求（OPTIONS）
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// setupRoutes 设置路由
func setupRoutes(s *AppServer) *gin.Engine {
	// 创建 Gin 引擎（带默认中间件）
	router := gin.Default()
	
	// 添加 CORS 中间件（放在所有路由之前）
	router.Use(corsMiddleware())
	
	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
	
	// MCP 相关路由
	// 这里添加你的 MCP 路由
	if s.mcpServer != nil {
		// 假设你的 MCP Server 有 HTTP 处理函数
		// 例如：router.POST("/mcp", gin.WrapH(s.mcpServer.Handler()))
	}
	
	// 其他业务路由
	// router.GET("/api/xxx", s.handleXXX)
	
	return router
}
