/*
 * Copyright 2022-2023 Chaos Meta Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmdexec

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ChaosMetaverse/chaosmetad/pkg/crclient"
	"github.com/ChaosMetaverse/chaosmetad/pkg/log"
	"github.com/ChaosMetaverse/chaosmetad/pkg/utils"
	"os/exec"
	"strings"
	"time"
)

const (
	InjectCheckInterval = time.Millisecond * 200
	execnsKey           = "chaosmeta_execns"
)

func StartSleepRecover(sleepTime int64, uid string) error {
	return StartBashCmd(utils.GetSleepRecoverCmd(sleepTime, uid))
}

func waitProExec(stdout, stderr *bytes.Buffer) (err error) {
	var msg, timer = "", time.NewTimer(InjectCheckInterval)
	for {
		<-timer.C
		if stderr.String() != "" || stdout.String() != "" {
			msg = stdout.String() + stderr.String()
			break
		}
		timer.Reset(InjectCheckInterval)
	}

	log.GetLogger().Debugf(msg)

	if strings.Index(msg, "error") >= 0 {
		return fmt.Errorf("inject error: %s", msg)
	}

	if strings.Index(msg, "[success]") >= 0 {
		return nil
	}

	return fmt.Errorf("unexpected output: %s", msg)
}

func SupportCmd(cmd string) bool {
	_, err := exec.LookPath(cmd)
	if err != nil {
		return false
	}

	return true
}

func RunBashCmdWithOutput(cmd string) ([]byte, error) {
	log.GetLogger().Debugf("run cmd with output: %s", cmd)
	return exec.Command("/bin/bash", "-c", cmd).CombinedOutput()
}

func RunBashCmdWithoutOutput(cmd string) error {
	log.GetLogger().Debugf("run cmd: %s", cmd)
	return exec.Command("/bin/bash", "-c", cmd).Run()
}

func StartBashCmd(cmd string) error {
	log.GetLogger().Debugf("start cmd: %s", cmd)
	return exec.Command("/bin/bash", "-c", cmd).Start()
}

func ExecContainer(cmd, cr, containerId, namespaces string) (int, error) {
	client, err := crclient.GetClient(cr)
	if err != nil {
		return utils.NoPid, fmt.Errorf("get cr[%s] client error: %s", cr, err.Error())
	}

	ctx := context.Background()
	targetPid, err := client.GetPidById(ctx, containerId)
	if err != nil {
		return utils.NoPid, fmt.Errorf("get pid of container[%s]'s init process error: %s", containerId, err.Error())
	}

	return StartBashCmdAndWaitPid(fmt.Sprintf("%s %d %s %s", utils.GetToolPath(execnsKey), targetPid, namespaces, cmd))
}

func StartBashCmdAndWaitPid(cmd string) (int, error) {
	log.GetLogger().Debugf("start cmd: %s", cmd)

	c := exec.Command("/bin/bash", "-c", cmd)
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr

	if err := c.Start(); err != nil {
		return utils.NoPid, fmt.Errorf("cmd start error: %s", err.Error())
	}

	if err := waitProExec(&stdout, &stderr); err != nil {
		return c.Process.Pid, fmt.Errorf("wait process exec error: %s", err.Error())
	}

	return c.Process.Pid, nil
}

func StartBashCmdAndWaitByUser(cmd, user string) error {
	log.GetLogger().Debugf("user: %s, start cmd: %s", user, cmd)

	c := exec.Command("runuser", "-l", user, "-c", cmd)
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr

	if err := c.Start(); err != nil {
		return fmt.Errorf("cmd start error: %s", err.Error())
	}

	if err := waitProExec(&stdout, &stderr); err != nil {
		return fmt.Errorf("wait process exec error: %s", err.Error())
	}

	return nil
}