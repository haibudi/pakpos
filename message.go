package pakpos

type MessageMonitor struct {
	Success int
	Fail    int
	Status  []string
}

func (m *MessageMonitor) Wait() {
}
