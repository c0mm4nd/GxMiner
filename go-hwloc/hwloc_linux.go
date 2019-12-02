package hwloc

//#cgo LDFLAGS: -lhwloc -lltdl -ldl -lnuma
//#cgo LDFLAGS: -static -static-libgcc
/*
#include <stdint.h>
#include <hwloc.h>
#if HWLOC_API_VERSION < 0x00010b00
#   define HWLOC_OBJ_NUMANODE HWLOC_OBJ_NODE
#endif

static void bindToNUMANode(uint32_t nodeId)
{

    hwloc_topology_t topology;
    hwloc_topology_init(&topology);
    hwloc_topology_load(topology);

    hwloc_obj_t node = hwloc_get_numanode_obj_by_os_index(topology, nodeId);
    if (node) {
#           if HWLOC_API_VERSION >= 0x20000
            hwloc_set_membind(topology, node->nodeset, HWLOC_MEMBIND_BIND, HWLOC_MEMBIND_THREAD | HWLOC_MEMBIND_BYNODESET);
#           else
            hwloc_set_membind_nodeset(topology, node->nodeset, HWLOC_MEMBIND_BIND, HWLOC_MEMBIND_THREAD);
#           endif

        //Platform::setThreadAffinity(static_cast<uint64_t>(hwloc_bitmap_first(node->cpuset))); // use cpuaffinity lib
    }

    hwloc_topology_destroy(topology);
}

*/
import "C"

func BindToNUMANode(affinity int64) {
	C.bindToNUMANode(C.uint32_t(affinity))
}
