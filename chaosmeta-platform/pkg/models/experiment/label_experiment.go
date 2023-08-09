package experiment

import (
	models "chaosmeta-platform/pkg/models/common"
	"github.com/beego/beego/v2/client/orm"
	"github.com/spf13/cast"
)

type LabelExperiment struct {
	ID             int    `json:"id,omitempty" orm:"pk;auto;column(id)"`
	LabelID        int    `json:"label_id" orm:"column(label_id);index"`
	ExperimentUUID string `json:"experiment_uuid" orm:"column(experiment_uuid);index"`
	models.BaseTimeModel
}

func (le *LabelExperiment) TableName() string {
	return TablePrefix + "label"
}

func (le *LabelExperiment) TableUnique() [][]string {
	return [][]string{{"label_id", "experiment_uuid"}}
}

func ListLabelIDsByExperimentUUID(experimentUUID string) ([]int, error) {
	o := models.GetORM()
	var (
		labelIDs    orm.ParamsList
		labelIDList []int
	)
	_, err := o.QueryTable(new(LabelExperiment).TableName()).Filter("experiment_uuid", experimentUUID).Distinct().ValuesFlat(&labelIDs, "label_id")
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, labelID := range labelIDs {
		labelIDList = append(labelIDList, cast.ToInt(labelID))
	}
	return labelIDList, nil
}

func AddLabelIDsToExperiment(experimentUUID string, labelIDs []int) error {
	o := models.GetORM()
	labelExperiments := make([]*LabelExperiment, len(labelIDs))
	for i, id := range labelIDs {
		labelExperiments[i] = &LabelExperiment{LabelID: id, ExperimentUUID: experimentUUID}
	}
	_, err := o.InsertMulti(len(labelExperiments), labelExperiments)
	return err
}

func ClearLabelIDsByExperimentUUID(experimentUUID string) error {
	o := models.GetORM()
	_, err := o.QueryTable(new(LabelExperiment).TableName()).Filter("experiment_uuid", experimentUUID).Delete()
	return err
}

func DeleteLabelIDsByExperimentUUIDAndLabelID(experimentUUID string, labelID int) error {
	o := models.GetORM()
	_, err := o.QueryTable(new(LabelExperiment).TableName()).Filter("experiment_uuid", experimentUUID).Filter("label_id", labelID).Delete()
	return err
}

func BatchSearchLabelExperiments(searchCriteria map[string]interface{}) ([]*LabelExperiment, error) {
	o := models.GetORM()
	labelExperiments := []*LabelExperiment{}
	qs := o.QueryTable(new(LabelExperiment).TableName())
	for key, value := range searchCriteria {
		qs = qs.Filter(key, value)
	}

	_, err := qs.All(&labelExperiments)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return labelExperiments, nil
}
