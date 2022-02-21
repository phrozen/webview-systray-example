package main

import (
	"context"
	"flag"
	"io/ioutil"
	"runtime"
	"time"

	"github.com/getlantern/systray"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/webview/webview"
)

var debug bool
var port string

func init() {
	flag.StringVar(&port, "p", "3000", "Port in which the server will run")
	flag.BoolVar(&debug, "d", false, "Debug mode for webview")
	flag.Parse()
}

// App ...
type App struct {
	server  *echo.Echo
	window  webview.WebView
	show    *systray.MenuItem
	control chan bool
	Log     echo.Logger
}

// Initialize ...
func Initialize() *App {
	app := new(App)
	app.control = make(chan bool)
	app.server = echo.New()
	app.server.Logger.SetLevel(log.INFO)
	app.Log = app.server.Logger
	app.server.Use(middleware.Logger())
	app.server.Use(middleware.Recover())
	app.server.File("/", "index.html")
	app.server.File("/favicon.ico", "icon.ico")
	return app
}

// Run ...
func (app *App) Run() {
	app.Log.Warn(app.server.Start(":" + port)) // blocks
	app.Log.Warn("Server Shutdown")
}

// OnReady ...
func (app *App) OnReady() {
	// Icon downloaded from
	// https://www.iconarchive.com/show/soft-scraps-icons-by-hopstarter/Gear-icon.html
	icon, err := ioutil.ReadFile("icon.ico")
	if err != nil {
		panic(err)
	}
	systray.SetTemplateIcon(icon, icon)
	app.show = systray.AddMenuItem("Show", "Shows/Creates Webview Window")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Exit", "Quit the app signalig webview to exit")
	go func() {
		for {
			select {
			case <-app.show.ClickedCh:
				app.control <- true
			case <-quit.ClickedCh:
				app.window.Terminate()
				close(app.control)
				systray.Quit()
			}
		}
	}()

}

// OnExit ...
func (app *App) OnExit() {
	app.Log.Info("Systray loop ended")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.server.Shutdown(ctx); err != nil {
		app.Log.Info(err)
	}
	app.Log.Warn("App Shutdown")
}

func main() {
	app := Initialize()

	go func() {
		app.Log.Info("App initialized running server...")
		app.Run()
	}()

	go func() {
		runtime.LockOSThread()
		for range app.control {
			app.show.Disable()
			app.Log.Info("Creating webview")
			app.window = webview.New(debug)
			app.window.SetTitle("Webview Systray Example")
			app.window.SetSize(800, 600, webview.HintNone)
			app.window.Navigate("http://localhost:" + port)
			app.window.Run()
			app.window.Destroy()
			app.Log.Info("Webview destroyed")
			app.show.Enable()
		}
	}()
	systray.Run(app.OnReady, app.OnExit)
}
