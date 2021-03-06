// Package pusher handles pushing to individual Cloud Foundry instances.
package pusher

import (
	"fmt"
	"io"

	I "github.com/compozed/deployadactyl/interfaces"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/go-errors/errors"
	"github.com/op/go-logging"
)

const (
	cannotDelete             = "cannot delete"
	cannotLogin              = "cannot login to"
	renamingApp              = "renaming app from %s to %s"
	renamedApp               = "renamed app from %s to %s"
	cannotRenameApp          = "cannot rename, app already exists"
	pushingNewApp            = "pushing new app %s to %s"
	newAppPath               = "using tempdir for app %s %s"
	mappingRoute             = "mapping route for %s to %s"
	deletedApp               = "deleted %s"
	rollingBackDeploy        = "rolling back deploy of %s"
	unableToDelete           = "unable to delete %s: %s"
	unableToRenameVenerable  = "unable to rename venerable app %s: %s"
	loggedIntoCloudFoundry   = "logged into cloud foundry %s"
	newAppDetected           = "new app detected"
	appRouteCreated          = "application route created at %s.%s"
	outputMessage            = "output from Cloud Foundry:\n"
	finishedPushSuccessfully = "finished push successfully on %s"
)

// Pusher has a courier used to push applications to Cloud Foundry.
type Pusher struct {
	Courier I.Courier
	Log     *logging.Logger
}

// Push pushes a single application to a Clound Foundry instance using blue green deployment.
// Blue green is done by renaming the current application to appName-venerable.
// Pushes the new application to the existing appName route with an included load balanced domain if provided.
//
// Returns Cloud Foundry logs if there is an error.
func (p Pusher) Push(appPath, domain string, deploymentInfo S.DeploymentInfo, out io.Writer) ([]byte, error) {
	renameOutput, err := p.Courier.Rename(deploymentInfo.AppName, deploymentInfo.AppName+"-venerable")
	if err != nil {
		if p.Courier.Exists(deploymentInfo.AppName) {
			p.Log.Errorf(cannotRenameApp)
			return nil, errors.New(string(renameOutput))
		}
		p.Log.Infof(newAppDetected)
	} else {
		p.Log.Infof(renamedApp, deploymentInfo.AppName, deploymentInfo.AppName+"-venerable")
	}

	p.Log.Infof(pushingNewApp, deploymentInfo.AppName, domain)
	p.Log.Debugf(newAppPath, deploymentInfo.AppName, appPath)
	pushOutput, err := p.Courier.Push(deploymentInfo.AppName, appPath, deploymentInfo.Instances)
	fmt.Fprint(out, string(pushOutput))
	if err != nil {
		logs, err := p.getCloudFoundryLogs(deploymentInfo.AppName)
		if err != nil {
			return logs, errors.New(err)
		}
		return logs, errors.New(outputMessage + string(pushOutput))
	}
	p.Log.Infof(outputMessage + string(pushOutput))

	p.Log.Debugf(mappingRoute, deploymentInfo.AppName, domain)
	mapRouteOutput, err := p.Courier.MapRoute(deploymentInfo.AppName, domain)
	fmt.Fprint(out, string(mapRouteOutput))
	if err != nil {
		logs, err := p.getCloudFoundryLogs(deploymentInfo.AppName)
		if err != nil {
			return logs, errors.New(string(pushOutput))
		}
		return logs, errors.New(err)
	}
	p.Log.Debugf(string(mapRouteOutput))
	p.Log.Infof(appRouteCreated, deploymentInfo.AppName, domain)
	return nil, nil
}

// DeleteVenerable will delete the venerable instance of your application.
func (p Pusher) DeleteVenerable(deploymentInfo S.DeploymentInfo, foundationURL string) error {
	venerableName := deploymentInfo.AppName + "-venerable"

	_, err := p.Courier.Delete(deploymentInfo.AppName + "-venerable")
	if err != nil {
		return errors.Errorf("%s %s: %s", cannotDelete, venerableName, err)
	}

	p.Log.Infof(deletedApp, venerableName)
	p.Log.Infof(finishedPushSuccessfully, foundationURL)

	return nil
}

// Rollback will rollback Push.
// Deletes the new application.
// Renames appName-venerable back to appName if this is not the first deploy.
func (p Pusher) Rollback(deploymentInfo S.DeploymentInfo, firstDeploy bool) error {
	p.Log.Errorf(rollingBackDeploy, deploymentInfo.AppName)
	venerableName := deploymentInfo.AppName + "-venerable"

	_, err := p.Courier.Delete(deploymentInfo.AppName)
	if err != nil {
		p.Log.Infof(unableToDelete, deploymentInfo.AppName, err)
	}
	p.Log.Infof(deletedApp, deploymentInfo.AppName)

	if !firstDeploy {
		_, err = p.Courier.Rename(venerableName, deploymentInfo.AppName)
		if err != nil {
			p.Log.Infof(unableToRenameVenerable, venerableName, err)
		}
		p.Log.Infof(renamedApp, venerableName, deploymentInfo.AppName)
	}

	return nil
}

// CleanUp removes the temporary directory created by the Executor.
func (p Pusher) CleanUp() error {
	return p.Courier.CleanUp()
}

// Login will login to a Cloud Foundry instance.
func (p Pusher) Login(foundationURL string, deploymentInfo S.DeploymentInfo, out io.Writer) error {
	p.Log.Debugf(
		`logging into cloud foundry with parameters:
		foundation URL: %+v
		username: %+v
		org: %+v
		space: %+v`,
		foundationURL, deploymentInfo.Username, deploymentInfo.Org, deploymentInfo.Space,
	)

	loginOutput, err := p.Courier.Login(
		foundationURL,
		deploymentInfo.Username,
		deploymentInfo.Password,
		deploymentInfo.Org,
		deploymentInfo.Space,
		deploymentInfo.SkipSSL,
	)
	out.Write(loginOutput)
	if err != nil {
		return errors.Errorf("%s %s: %s", cannotLogin, foundationURL, err)
	}
	p.Log.Infof(loggedIntoCloudFoundry, foundationURL)

	return nil
}

// Exists uses the courier to check if the application exists.
func (p Pusher) Exists(appName string) bool {
	return p.Courier.Exists(appName)
}

func (p Pusher) getCloudFoundryLogs(appName string) ([]byte, error) {
	return p.Courier.Logs(appName)
}
