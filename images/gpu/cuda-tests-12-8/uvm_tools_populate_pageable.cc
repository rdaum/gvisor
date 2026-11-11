// Copyright 2026 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include <fcntl.h>
#include <sys/ioctl.h>
#include <unistd.h>

#include <cerrno>
#include <cstdint>
#include <cstdio>
#include <cstring>

namespace {

// NVIDIA's UVM ioctl numbers are independent of _IOC encoding.
constexpr unsigned long kUVMPopulatePageable = 71;
struct PopulateParams {
  uint64_t base;
  uint64_t length;
  uint32_t flags;
  uint32_t status;
};
static_assert(sizeof(PopulateParams) == 24);

// The tools device does not implement UVM_POPULATE_PAGEABLE. The driver
// rejects the command by number, before it reads the parameters.
bool ExpectInvalid(int fd, void* params, const char* description) {
  errno = 0;
  const int result = ioctl(fd, kUVMPopulatePageable, params);
  if (result != -1 || errno != EINVAL) {
    std::fprintf(stderr, "FAIL: %s: returned %d, errno=%d (%s); want -1, %s\n",
                 description, result, errno, std::strerror(errno),
                 std::strerror(EINVAL));
    return false;
  }
  return true;
}

}  // namespace

int main() {
  std::setvbuf(stdout, nullptr, _IONBF, 0);
  const int fd = open("/dev/nvidia-uvm-tools", O_RDWR | O_CLOEXEC);
  if (fd < 0) {
    std::perror("FAIL: open /dev/nvidia-uvm-tools");
    return 1;
  }
  PopulateParams params{0, 0, 0, UINT32_MAX};
  if (!ExpectInvalid(fd, &params, "readable parameters")) {
    return 1;
  }
  if (!ExpectInvalid(fd, reinterpret_cast<void*>(1), "unreadable parameters")) {
    return 1;
  }
  if (close(fd) != 0) {
    std::perror("FAIL: close /dev/nvidia-uvm-tools");
    return 1;
  }
  std::puts("PASS: UVM_POPULATE_PAGEABLE is invalid on the UVM tools device");
  return 0;
}
