package controllers

import (
	"fmt"
	"net/http"

	"github.com/astaxie/beego/logs"
	"github.com/niyanchun/k8s-middleware/models"
)

// Operations about Apps
type AppController struct {
	BaseController
	namespace string
}

func (app *AppController) Prepare() {
	method := app.Ctx.Input.Method()
	if method == http.MethodPost || method == http.MethodPut {
		return
	}

	namespace := app.GetString("namespace")
	app.CheckEmpty(namespace, "namespace")
	app.namespace = namespace
}

// @Description list user's all apps
// @Param namespace query string true "namespace"
// @router / [get]
func (app *AppController) List() {
	svc_list, err := models.Client.ListServices(app.namespace)
	app.CheckError(err, "[app.List] list service error", http.StatusInternalServerError)

	var apps []models.AppInfo
	var a models.AppInfo
	for _, svc := range svc_list.Items {
		a.Name = svc.Name

		err, status := models.GetPodsStatus(svc.Namespace, svc.Labels)
		if err != nil {
			s := fmt.Sprintf("get service %s's status error, %s", svc.Name, err.Error())
			logs.Error(s)
			a.Status = models.AppStatusUnkonwn
		} else {
			a.Status = status
		}

		apps = append(apps, a)
	}

	app.Data["json"] = apps
	app.ServeJSON()
}

// @Description get app detail: service + RC + Pod
// @Param namespace query string true "namespace"
// @Param app_name path string true "app name"
// @router /:app_name [get]
func (app *AppController) Get() {
	app_name := app.GetString(":app_name")
	app.CheckEmpty(app_name, "app_name")

	var app_details models.AppDetails

	service, err := models.Client.GetService(app.namespace, app_name)
	app.CheckError(err, "get service error", http.StatusInternalServerError)
	app_details.Service = *service

	rc, err := models.Client.GetReplicationController(app.namespace, app_name)
	app.CheckError(err, "get rc error", http.StatusInternalServerError)
	app_details.ReplicationController = *rc

	labels := service.Labels
	//pod, err := models.Client.GetPod(app.namespace, app_name)
	pods, err := models.Client.ListPodsWithLabel(app.namespace, labels)
	app.CheckError(err, "get pod error", http.StatusInternalServerError)

	pod_num := len(pods.Items)
	if pod_num == 1 {
		app_details.Pod = pods.Items[0]
	} else if pod_num == 0 {
		// no pod is running, just set App status as "Stopped"
		app_details.Pod.Status.Phase = models.AppStatusStopped
	} else {
		s := fmt.Sprintf("pods with label %v is not unique", labels)
		logs.Error(s)
		app.CustomAbort(http.StatusInternalServerError, s)
	}

	app.Data["json"] = app_details
	app.ServeJSON()
}