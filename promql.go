package promql

import (
	"fmt"
	"strings"
)

type Stringer interface {
	String() string
}

type EmptyString struct{}

func (es EmptyString) Stringer() string { return "" }

type Node interface {
	Stringer
	Self() string
	Children() []Node
}

type ConstantStringNode struct {
	constant string
}

func NewConstantStringNode(constantString string) ConstantStringNode {
	return ConstantStringNode{constant: constantString}
}

func (m ConstantStringNode) String() string {
	return m.Self()
}

func (m ConstantStringNode) Self() string {
	return m.constant
}

func (m ConstantStringNode) Children() []Node {
	return nil
}

var _ Node = (*ConstantStringNode)(nil)

// time series selector
type TSSelector struct {
	Name     string
	Labels   []Label
	duration string // 可选, 比如5m
	offset   string // 可选，比如offset 5m
}

var _ Node = (*TSSelector)(nil)

func (m TSSelector) String() string {
	return m.Self()
}

func (m TSSelector) Self() string {
	if m.Name == "" && len(m.Labels) == 0 {
		panic("metric name and labels cannot be both empty")
	}
	s := m.Name
	if len(m.Labels) != 0 {
		labelStrings := make([]string, 0, len(m.Labels))
		for _, label := range m.Labels {
			labelStrings = append(labelStrings, label.Stringer())
		}
		s += fmt.Sprintf("{%s}", strings.Join(labelStrings, ", "))
	}
	if m.duration != "" {
		s += fmt.Sprintf("[%s]", m.duration)
	}
	if m.offset != "" {
		s += fmt.Sprintf(" offset %s", m.offset)
	}
	return s
}

func (m TSSelector) Children() []Node {
	return nil
}

func (m TSSelector) WithLabels(labels ...Label) TSSelector {
	m.Labels = append(m.Labels, labels...)
	return m
}

func (m TSSelector) WithDuration(duration string) TSSelector {
	m.duration = duration
	return m
}

func (m TSSelector) WithOffset(offset string) TSSelector {
	m.offset = offset
	return m
}

type Label struct {
	Key     string
	Value   string
	Matcher string // = != =~
}

func (l Label) Stringer() string {
	return fmt.Sprintf(`%s%s"%s"`, l.Key, l.Matcher, l.Value)
}

// 函数， 比如 rate
type Func struct {
	fun        string
	parameters []Node // 长度不定， 1， 2，等
}

func NewFunc(fun string, parameters ...Node) Func {
	return Func{fun: fun, parameters: parameters}
}

func (f Func) WithParameters(params ...Node) Func {
	f.parameters = append(f.parameters, params...)
	return f
}

var _ Node = (*Func)(nil)

func (f Func) String() string {
	params := make([]string, 0, len(f.parameters))
	for _, p := range f.parameters {
		params = append(params, p.String())
	}
	return fmt.Sprintf("%s(%s)", f.Self(), strings.Join(params, ", "))
}

func (f Func) Self() string {
	return f.fun
}

func (f Func) Children() []Node {
	return f.parameters
}

// 二元操作符
type BinaryOp struct {
	operator string         // + - * / == != > < >= <= and or unless
	operands []Node         // 长度为2
	matcher  *VectorMatcher // 可选
}

func NewBinaryOp(operator string) BinaryOp {
	return BinaryOp{operator: operator}
}

func (bo BinaryOp) WithOperands(left, right Node) BinaryOp {
	bo.operands = []Node{left, right}
	return bo
}

func (bo BinaryOp) WithMatcher(vm VectorMatcher) BinaryOp {
	bo.matcher = &vm
	return bo
}

var _ Node = (*BinaryOp)(nil)

func (bo BinaryOp) String() string {
	if _, ok := bo.operands[0].(BinaryOp); ok {
		return fmt.Sprintf("(%s) %s %s", bo.operands[0].String(), bo.Self(), bo.operands[1].String())
	}
	if _, ok := bo.operands[1].(BinaryOp); ok {
		return fmt.Sprintf("%s %s (%s)", bo.operands[0].String(), bo.Self(), bo.operands[1].String())
	}
	return fmt.Sprintf("%s %s %s", bo.operands[0].String(), bo.Self(), bo.operands[1].String())
}

func (bo BinaryOp) Self() string {
	s := bo.operator
	if bo.matcher != nil {
		s += " " + bo.matcher.String()
	}
	return s
}

func (bo BinaryOp) Children() []Node {
	return bo.operands
}

type VectorMatcher struct {
	keyword string // on/ignoring
	labels  []string
	group   *GroupModifier // 可选
}

func NewVectorMatcher(keyword string, labels ...string) VectorMatcher {
	return VectorMatcher{keyword: keyword, labels: labels}
}

func (vm VectorMatcher) String() string {
	s := fmt.Sprintf("%s(%s)", vm.keyword, strings.Join(vm.labels, ", "))
	if vm.group != nil {
		s += " " + vm.group.String()
	}
	return s
}

func (vm VectorMatcher) WithLabels(labels ...string) VectorMatcher {
	vm.labels = append(vm.labels, labels...)
	return vm
}

func (vm VectorMatcher) WithGroupLeft(labels ...string) VectorMatcher {
	vm.group = &GroupModifier{left: true, labels: labels}
	return vm
}

func (vm VectorMatcher) WithGroupRight(labels ...string) VectorMatcher {
	vm.group = &GroupModifier{left: false, labels: labels}
	return vm
}

type GroupModifier struct {
	left   bool
	labels []string
}

func (gm GroupModifier) String() string {
	var group string
	if gm.left {
		group = "group_left"
	} else {
		group = "group_right"
	}
	if len(gm.labels) == 0 {
		return group
	}
	return fmt.Sprintf("%s(%s)", group, strings.Join(gm.labels, ", "))
}

// 聚合操作符
type AggregationOp struct {
	operator  string             // sum, min, max, avg, topk, count, quantile ...
	operand   Node               // aggregate a single instant vector
	clause    *AggregationClause // 可选，比如 by(code)
	parameter *Scalar            // only required for count_values, quantile, topk and bottomk.
}

func NewAggregationOp(operator string) AggregationOp {
	return AggregationOp{operator: operator}
}

func (ao AggregationOp) SetOperand(operand Node) AggregationOp {
	ao.operand = operand
	return ao
}

func (ao AggregationOp) WithByClause(labels ...string) AggregationOp {
	ao.clause = &AggregationClause{keyword: "by", labels: labels}
	return ao
}

func (ao AggregationOp) WithWithoutClause(labels ...string) AggregationOp {
	ao.clause = &AggregationClause{keyword: "without", labels: labels}
	return ao
}

func (ao AggregationOp) WithClause(keyword string, labels ...string) AggregationOp {
	ao.clause = &AggregationClause{keyword: keyword, labels: labels}
	return ao
}

func (ao AggregationOp) WithParameter(param Scalar) AggregationOp {
	ao.parameter = &param
	return ao
}

var _ Node = (*AggregationOp)(nil)

func (ao AggregationOp) Self() string {
	s := ao.operator
	if ao.clause != nil {
		s += " " + ao.clause.String()
	}
	return s
}

func (ao AggregationOp) String() string {
	if ao.parameter != nil {
		return fmt.Sprintf("%s (%f, %s)", ao.Self(), *ao.parameter, ao.operand.String())
	}
	return fmt.Sprintf("%s (%s)", ao.Self(), ao.operand.String())
}

func (ao AggregationOp) Children() []Node {
	return []Node{ao.operand}
}

type AggregationClause struct {
	keyword string // by, without
	labels  []string
}

func (ac AggregationClause) String() string {
	return fmt.Sprintf("%s (%s)", ac.keyword, strings.Join(ac.labels, ", "))
}

// 浮点数标量
type Scalar float64

var _ Node = (*Scalar)(nil)

func (s Scalar) String() string {
	return s.Self()
}

// TODO: 妥善处理显示的精度
func (s Scalar) Self() string {
	return fmt.Sprintf("%.4f", s)
}

func (s Scalar) Children() []Node {
	return nil
}
