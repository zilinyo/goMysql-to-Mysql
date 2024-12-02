/*
 * Copyright 2020-2021 the original author(https://github.com/wj596)
 *
 * <p>
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
 * </p>
 */
package main

import (
	"cardappcanal/cmd"
)

var (
	helpFlag     bool
	cfgPath      string
	stockFlag    bool
	positionFlag bool
	statusFlag   bool
)

//func init() {
//	flag.BoolVar(&helpFlag, "help", false, "this help")
//	flag.StringVar(&cfgPath, "config", "app.yml", "application config file")
//	flag.BoolVar(&stockFlag, "stock", false, "stock data import")
//	flag.BoolVar(&positionFlag, "position", false, "set dump position")
//	flag.BoolVar(&statusFlag, "status", false, "display application status")
//	flag.Usage = usage
//}

func main() {
	cmd.Exec()
}
