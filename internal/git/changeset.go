package git

type ChangeSet struct {
	Created []string `yaml:"created" json:"created"`
	Updated []string `yaml:"updated" json:"updated"`
	Deleted []string `yaml:"deleted" json:"deleted"`
}

func NewChangeSet(created []string, updated []string, deleted []string) *ChangeSet {
	return &ChangeSet{
		Created: created,
		Updated: updated,
		Deleted: deleted}
}

func (changeSet *ChangeSet) Len() int {
	return len(changeSet.Created) + len(changeSet.Updated) + len(changeSet.Updated)
}
