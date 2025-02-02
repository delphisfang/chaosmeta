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

package recoverhandler

import (
	"context"
	"fmt"
	"github.com/agiledragon/gomonkey"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/traas-stack/chaosmeta/chaosmeta-inject-operator/api/v1alpha1"
	mockscopehandler "github.com/traas-stack/chaosmeta/chaosmeta-inject-operator/mock/scopehandler"
	"github.com/traas-stack/chaosmeta/chaosmeta-inject-operator/pkg/common"
	"github.com/traas-stack/chaosmeta/chaosmeta-inject-operator/pkg/model"
	"github.com/traas-stack/chaosmeta/chaosmeta-inject-operator/pkg/scopehandler"
	"testing"
	"time"
)

func TestRecoverPhaseHandler_SolveCreated_OneToRunning(t *testing.T) {
	// init data
	var (
		ctx     = context.Background()
		nowTime = time.Now().Format(model.TimeFormat)
		exp     = &v1alpha1.Experiment{
			Spec: v1alpha1.ExperimentSpec{
				Scope: v1alpha1.PodScopeType,
				RangeMode: &v1alpha1.RangeMode{
					Type:  v1alpha1.CountRangeType,
					Value: 3,
				},
				Experiment: &v1alpha1.ExperimentCommon{
					Duration: "2m",
					Target:   "cpu",
					Fault:    "burn",
					Args: []v1alpha1.ArgsUnit{
						{
							Key:       "percent",
							Value:     "90",
							ValueType: v1alpha1.IntVType,
						},
						{
							Key:   v1alpha1.ContainerKey,
							Value: "nginx",
						},
					},
				},
				Selector: []v1alpha1.SelectorUnit{
					{
						Namespace: "chaosmeta",
					},
				},
				TargetPhase: v1alpha1.RecoverPhaseType,
			},
			Status: v1alpha1.ExperimentStatus{
				Phase:      v1alpha1.RecoverPhaseType,
				Status:     v1alpha1.CreatedStatusType,
				CreateTime: nowTime,
				UpdateTime: nowTime,
				Detail: v1alpha1.ExperimentDetail{
					Inject: []v1alpha1.ExperimentDetailUnit{
						{
							InjectObjectName: "pod/chaosmeta/chaosmeta-0",
							UID:              "fwaf",
							Status:           v1alpha1.SuccessStatusType,
						},
					},
					Recover: []v1alpha1.ExperimentDetailUnit{
						{
							InjectObjectName: "pod/chaosmeta/chaosmeta-0",
							UID:              "fwaf",
							Status:           v1alpha1.CreatedStatusType,
						},
					},
				},
			},
		}
		reContainer = &model.PodObject{

			Namespace: "chaosmeta",
			PodName:   "chaosmeta-0",
			PodUID:    "d32tg32",
			PodIP:     "1.2.3.4",
			NodeName:  "node-1",
			NodeIP:    "2.2.2.2",

			ContainerID:      "g3g3g",
			ContainerRuntime: "docker",
		}
		re = model.AtomicObject(reContainer)
	)
	common.SetGoroutinePool(5)

	// mock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	scopeHandlerMock := mockscopehandler.NewMockScopeHandler(ctrl)
	scopeHandlerMock.EXPECT().GetInjectObject(ctx, exp.Spec.Experiment, reContainer.GetObjectName()).Return(re, nil)
	scopeHandlerMock.EXPECT().ExecuteRecover(ctx, re, exp.Status.Detail.Inject[0].UID, "", exp.Spec.Experiment).Return(nil)
	gomonkey.ApplyFunc(scopehandler.GetScopeHandler, func(v1alpha1.ScopeType) scopehandler.ScopeHandler {
		return scopeHandlerMock
	})

	// execute test
	phaseHandler := RecoverPhaseHandler{}
	assert.Equal(t, 0, common.GetGoroutinePool().GetLen())
	phaseHandler.SolveCreated(ctx, exp)
	assert.Equal(t, 0, common.GetGoroutinePool().GetLen())

	// check result

	assert.Equal(t, v1alpha1.RunningStatusType, exp.Status.Status)
	assert.Equal(t, v1alpha1.RunningStatusType, exp.Status.Detail.Recover[0].Status)
}

func TestRecoverPhaseHandler_SolveRunning(t *testing.T) {
	// init data
	var (
		ctx     = context.Background()
		nowTime = time.Now().Format(model.TimeFormat)
		exp     = &v1alpha1.Experiment{
			Spec: v1alpha1.ExperimentSpec{
				Scope: v1alpha1.PodScopeType,
				RangeMode: &v1alpha1.RangeMode{
					Type:  v1alpha1.CountRangeType,
					Value: 3,
				},
				Experiment: &v1alpha1.ExperimentCommon{
					Duration: "2m",
					Target:   "cpu",
					Fault:    "burn",
					Args: []v1alpha1.ArgsUnit{
						{
							Key:       "percent",
							Value:     "90",
							ValueType: v1alpha1.IntVType,
						},
						{
							Key:   v1alpha1.ContainerKey,
							Value: "nginx",
						},
					},
				},
				Selector: []v1alpha1.SelectorUnit{
					{
						Namespace: "chaosmeta",
					},
				},
				TargetPhase: v1alpha1.InjectPhaseType,
			},
			Status: v1alpha1.ExperimentStatus{
				Phase:      v1alpha1.RecoverPhaseType,
				Status:     v1alpha1.RunningStatusType,
				CreateTime: nowTime,
				UpdateTime: nowTime,
				Detail: v1alpha1.ExperimentDetail{
					Inject: []v1alpha1.ExperimentDetailUnit{
						{
							InjectObjectName: "pod/chaosmeta/chaosmeta-1",
							UID:              "fwaf1",
							Status:           v1alpha1.SuccessStatusType,
						},
						{
							InjectObjectName: "pod/chaosmeta/chaosmeta-2",
							UID:              "fwaf2",
							Status:           v1alpha1.SuccessStatusType,
						},
					},
					Recover: []v1alpha1.ExperimentDetailUnit{
						{
							InjectObjectName: "pod/chaosmeta/chaosmeta-1",
							UID:              "fwaf1",
							Status:           v1alpha1.RunningStatusType,
						},
						{
							InjectObjectName: "pod/chaosmeta/chaosmeta-2",
							UID:              "fwaf2",
							Status:           v1alpha1.RunningStatusType,
						},
					},
				},
			},
		}
		reContainer1 = &model.PodObject{

			Namespace: "chaosmeta",
			PodName:   "chaosmeta-1",
			PodUID:    "d32tg31",
			PodIP:     "1.2.3.1",
			NodeName:  "node-1",
			NodeIP:    "2.2.2.1",

			ContainerID:      "g3g3g1",
			ContainerRuntime: "docker",
		}
		reContainer2 = &model.PodObject{

			Namespace: "chaosmeta",
			PodName:   "chaosmeta-2",
			PodUID:    "d32tg32",
			PodIP:     "1.2.3.2",
			NodeName:  "node-2",
			NodeIP:    "2.2.2.2",

			ContainerID:      "g3g3g2",
			ContainerRuntime: "docker",
		}
		re1 = model.AtomicObject(reContainer1)
		re2 = model.AtomicObject(reContainer2)
	)
	common.SetGoroutinePool(5)

	// mock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	scopeHandlerMock := mockscopehandler.NewMockScopeHandler(ctrl)
	scopeHandlerMock.EXPECT().GetInjectObject(ctx, exp.Spec.Experiment, reContainer1.GetObjectName()).Return(re1, nil)
	scopeHandlerMock.EXPECT().GetInjectObject(ctx, exp.Spec.Experiment, reContainer2.GetObjectName()).Return(re2, nil)
	scopeHandlerMock.EXPECT().QueryExperiment(ctx, re1, exp.Status.Detail.Inject[0].UID, "", exp.Spec.Experiment, v1alpha1.RecoverPhaseType).Return(nil, fmt.Errorf("expected fail"))
	scopeHandlerMock.EXPECT().QueryExperiment(ctx, re2, exp.Status.Detail.Inject[1].UID, "", exp.Spec.Experiment, v1alpha1.RecoverPhaseType).Return(nil, fmt.Errorf("expected fail"))

	gomonkey.ApplyFunc(scopehandler.GetScopeHandler, func(v1alpha1.ScopeType) scopehandler.ScopeHandler {
		return scopeHandlerMock
	})

	// execute test
	phaseHandler := RecoverPhaseHandler{}
	assert.Equal(t, 0, common.GetGoroutinePool().GetLen())
	phaseHandler.SolveRunning(ctx, exp)
	assert.Equal(t, 0, common.GetGoroutinePool().GetLen())

	// check result

	assert.Equal(t, v1alpha1.FailedStatusType, exp.Status.Status)
	assert.Equal(t, v1alpha1.FailedStatusType, exp.Status.Detail.Recover[0].Status)
	assert.Equal(t, v1alpha1.FailedStatusType, exp.Status.Detail.Recover[1].Status)
}
