package manager

type SlaveConfig struct {
	Enable bool
	Master string
	Key    string
}

func (m *Manager) AsSlave() {

}
