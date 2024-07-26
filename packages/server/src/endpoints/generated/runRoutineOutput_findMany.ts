export const runRoutineOutput_findMany = {
  "fieldName": "runRoutineOutputs",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runRoutineOutputs",
        "loc": {
          "start": 839,
          "end": 856
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 857,
              "end": 862
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 865,
                "end": 870
              }
            },
            "loc": {
              "start": 864,
              "end": 870
            }
          },
          "loc": {
            "start": 857,
            "end": 870
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "edges",
              "loc": {
                "start": 878,
                "end": 883
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "cursor",
                    "loc": {
                      "start": 894,
                      "end": 900
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 894,
                    "end": 900
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 909,
                      "end": 913
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 928,
                            "end": 930
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 928,
                          "end": 930
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "data",
                          "loc": {
                            "start": 943,
                            "end": 947
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 943,
                          "end": 947
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "output",
                          "loc": {
                            "start": 960,
                            "end": 966
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 985,
                                  "end": 987
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 985,
                                "end": 987
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 1004,
                                  "end": 1009
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1004,
                                "end": 1009
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 1026,
                                  "end": 1030
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1026,
                                "end": 1030
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "routineVersion",
                                "loc": {
                                  "start": 1047,
                                  "end": 1061
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1084,
                                        "end": 1086
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1084,
                                      "end": 1086
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "complexity",
                                      "loc": {
                                        "start": 1107,
                                        "end": 1117
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1107,
                                      "end": 1117
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAutomatable",
                                      "loc": {
                                        "start": 1138,
                                        "end": 1151
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1138,
                                      "end": 1151
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 1172,
                                        "end": 1182
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1172,
                                      "end": 1182
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isDeleted",
                                      "loc": {
                                        "start": 1203,
                                        "end": 1212
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1203,
                                      "end": 1212
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isLatest",
                                      "loc": {
                                        "start": 1233,
                                        "end": 1241
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1233,
                                      "end": 1241
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isPrivate",
                                      "loc": {
                                        "start": 1262,
                                        "end": 1271
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1262,
                                      "end": 1271
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "root",
                                      "loc": {
                                        "start": 1292,
                                        "end": 1296
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1323,
                                              "end": 1325
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1323,
                                            "end": 1325
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isInternal",
                                            "loc": {
                                              "start": 1350,
                                              "end": 1360
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1350,
                                            "end": 1360
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isPrivate",
                                            "loc": {
                                              "start": 1385,
                                              "end": 1394
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1385,
                                            "end": 1394
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1297,
                                        "end": 1416
                                      }
                                    },
                                    "loc": {
                                      "start": 1292,
                                      "end": 1416
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "routineType",
                                      "loc": {
                                        "start": 1437,
                                        "end": 1448
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1437,
                                      "end": 1448
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 1469,
                                        "end": 1481
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1508,
                                              "end": 1510
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1508,
                                            "end": 1510
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 1535,
                                              "end": 1543
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1535,
                                            "end": 1543
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1568,
                                              "end": 1579
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1568,
                                            "end": 1579
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "instructions",
                                            "loc": {
                                              "start": 1604,
                                              "end": 1616
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1604,
                                            "end": 1616
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1641,
                                              "end": 1645
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1641,
                                            "end": 1645
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1482,
                                        "end": 1667
                                      }
                                    },
                                    "loc": {
                                      "start": 1469,
                                      "end": 1667
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "versionIndex",
                                      "loc": {
                                        "start": 1688,
                                        "end": 1700
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1688,
                                      "end": 1700
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "versionLabel",
                                      "loc": {
                                        "start": 1721,
                                        "end": 1733
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1721,
                                      "end": 1733
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1062,
                                  "end": 1751
                                }
                              },
                              "loc": {
                                "start": 1047,
                                "end": 1751
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "standardVersion",
                                "loc": {
                                  "start": 1768,
                                  "end": 1783
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1806,
                                        "end": 1808
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1806,
                                      "end": 1808
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 1829,
                                        "end": 1839
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1829,
                                      "end": 1839
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 1860,
                                        "end": 1870
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1860,
                                      "end": 1870
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 1891,
                                        "end": 1901
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1891,
                                      "end": 1901
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isFile",
                                      "loc": {
                                        "start": 1922,
                                        "end": 1928
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1922,
                                      "end": 1928
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isLatest",
                                      "loc": {
                                        "start": 1949,
                                        "end": 1957
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1949,
                                      "end": 1957
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isPrivate",
                                      "loc": {
                                        "start": 1978,
                                        "end": 1987
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1978,
                                      "end": 1987
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "default",
                                      "loc": {
                                        "start": 2008,
                                        "end": 2015
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2008,
                                      "end": 2015
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "standardType",
                                      "loc": {
                                        "start": 2036,
                                        "end": 2048
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2036,
                                      "end": 2048
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "props",
                                      "loc": {
                                        "start": 2069,
                                        "end": 2074
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2069,
                                      "end": 2074
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "yup",
                                      "loc": {
                                        "start": 2095,
                                        "end": 2098
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2095,
                                      "end": 2098
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "versionIndex",
                                      "loc": {
                                        "start": 2119,
                                        "end": 2131
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2119,
                                      "end": 2131
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "versionLabel",
                                      "loc": {
                                        "start": 2152,
                                        "end": 2164
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2152,
                                      "end": 2164
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "commentsCount",
                                      "loc": {
                                        "start": 2185,
                                        "end": 2198
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2185,
                                      "end": 2198
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "directoryListingsCount",
                                      "loc": {
                                        "start": 2219,
                                        "end": 2241
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2219,
                                      "end": 2241
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "forksCount",
                                      "loc": {
                                        "start": 2262,
                                        "end": 2272
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2262,
                                      "end": 2272
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reportsCount",
                                      "loc": {
                                        "start": 2293,
                                        "end": 2305
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2293,
                                      "end": 2305
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 2326,
                                        "end": 2329
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canComment",
                                            "loc": {
                                              "start": 2356,
                                              "end": 2366
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2356,
                                            "end": 2366
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canCopy",
                                            "loc": {
                                              "start": 2391,
                                              "end": 2398
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2391,
                                            "end": 2398
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 2423,
                                              "end": 2432
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2423,
                                            "end": 2432
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canReport",
                                            "loc": {
                                              "start": 2457,
                                              "end": 2466
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2457,
                                            "end": 2466
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 2491,
                                              "end": 2500
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2491,
                                            "end": 2500
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUse",
                                            "loc": {
                                              "start": 2525,
                                              "end": 2531
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2525,
                                            "end": 2531
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 2556,
                                              "end": 2563
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2556,
                                            "end": 2563
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2330,
                                        "end": 2585
                                      }
                                    },
                                    "loc": {
                                      "start": 2326,
                                      "end": 2585
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "root",
                                      "loc": {
                                        "start": 2606,
                                        "end": 2610
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2637,
                                              "end": 2639
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2637,
                                            "end": 2639
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 2664,
                                              "end": 2674
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2664,
                                            "end": 2674
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 2699,
                                              "end": 2709
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2699,
                                            "end": 2709
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isPrivate",
                                            "loc": {
                                              "start": 2734,
                                              "end": 2743
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2734,
                                            "end": 2743
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "issuesCount",
                                            "loc": {
                                              "start": 2768,
                                              "end": 2779
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2768,
                                            "end": 2779
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "labels",
                                            "loc": {
                                              "start": 2804,
                                              "end": 2810
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "FragmentSpread",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "Label_list",
                                                  "loc": {
                                                    "start": 2844,
                                                    "end": 2854
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 2841,
                                                  "end": 2854
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2811,
                                              "end": 2880
                                            }
                                          },
                                          "loc": {
                                            "start": 2804,
                                            "end": 2880
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "owner",
                                            "loc": {
                                              "start": 2905,
                                              "end": 2910
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "InlineFragment",
                                                "typeCondition": {
                                                  "kind": "NamedType",
                                                  "name": {
                                                    "kind": "Name",
                                                    "value": "Team",
                                                    "loc": {
                                                      "start": 2948,
                                                      "end": 2952
                                                    }
                                                  },
                                                  "loc": {
                                                    "start": 2948,
                                                    "end": 2952
                                                  }
                                                },
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "FragmentSpread",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "Team_nav",
                                                        "loc": {
                                                          "start": 2990,
                                                          "end": 2998
                                                        }
                                                      },
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2987,
                                                        "end": 2998
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2953,
                                                    "end": 3028
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2941,
                                                  "end": 3028
                                                }
                                              },
                                              {
                                                "kind": "InlineFragment",
                                                "typeCondition": {
                                                  "kind": "NamedType",
                                                  "name": {
                                                    "kind": "Name",
                                                    "value": "User",
                                                    "loc": {
                                                      "start": 3064,
                                                      "end": 3068
                                                    }
                                                  },
                                                  "loc": {
                                                    "start": 3064,
                                                    "end": 3068
                                                  }
                                                },
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "FragmentSpread",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "User_nav",
                                                        "loc": {
                                                          "start": 3106,
                                                          "end": 3114
                                                        }
                                                      },
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3103,
                                                        "end": 3114
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 3069,
                                                    "end": 3144
                                                  }
                                                },
                                                "loc": {
                                                  "start": 3057,
                                                  "end": 3144
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2911,
                                              "end": 3170
                                            }
                                          },
                                          "loc": {
                                            "start": 2905,
                                            "end": 3170
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "permissions",
                                            "loc": {
                                              "start": 3195,
                                              "end": 3206
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3195,
                                            "end": 3206
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "questionsCount",
                                            "loc": {
                                              "start": 3231,
                                              "end": 3245
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3231,
                                            "end": 3245
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "score",
                                            "loc": {
                                              "start": 3270,
                                              "end": 3275
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3270,
                                            "end": 3275
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 3300,
                                              "end": 3309
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3300,
                                            "end": 3309
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tags",
                                            "loc": {
                                              "start": 3334,
                                              "end": 3338
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "FragmentSpread",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "Tag_list",
                                                  "loc": {
                                                    "start": 3372,
                                                    "end": 3380
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 3369,
                                                  "end": 3380
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3339,
                                              "end": 3406
                                            }
                                          },
                                          "loc": {
                                            "start": 3334,
                                            "end": 3406
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "transfersCount",
                                            "loc": {
                                              "start": 3431,
                                              "end": 3445
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3431,
                                            "end": 3445
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "views",
                                            "loc": {
                                              "start": 3470,
                                              "end": 3475
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3470,
                                            "end": 3475
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 3500,
                                              "end": 3503
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canDelete",
                                                  "loc": {
                                                    "start": 3534,
                                                    "end": 3543
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3534,
                                                  "end": 3543
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canBookmark",
                                                  "loc": {
                                                    "start": 3572,
                                                    "end": 3583
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3572,
                                                  "end": 3583
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canTransfer",
                                                  "loc": {
                                                    "start": 3612,
                                                    "end": 3623
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3612,
                                                  "end": 3623
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canUpdate",
                                                  "loc": {
                                                    "start": 3652,
                                                    "end": 3661
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3652,
                                                  "end": 3661
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canRead",
                                                  "loc": {
                                                    "start": 3690,
                                                    "end": 3697
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3690,
                                                  "end": 3697
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canReact",
                                                  "loc": {
                                                    "start": 3726,
                                                    "end": 3734
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3726,
                                                  "end": 3734
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 3763,
                                                    "end": 3775
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3763,
                                                  "end": 3775
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isViewed",
                                                  "loc": {
                                                    "start": 3804,
                                                    "end": 3812
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3804,
                                                  "end": 3812
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reaction",
                                                  "loc": {
                                                    "start": 3841,
                                                    "end": 3849
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3841,
                                                  "end": 3849
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3504,
                                              "end": 3875
                                            }
                                          },
                                          "loc": {
                                            "start": 3500,
                                            "end": 3875
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2611,
                                        "end": 3897
                                      }
                                    },
                                    "loc": {
                                      "start": 2606,
                                      "end": 3897
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 3918,
                                        "end": 3930
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3957,
                                              "end": 3959
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3957,
                                            "end": 3959
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 3984,
                                              "end": 3992
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3984,
                                            "end": 3992
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4017,
                                              "end": 4028
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4017,
                                            "end": 4028
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "jsonVariable",
                                            "loc": {
                                              "start": 4053,
                                              "end": 4065
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4053,
                                            "end": 4065
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4090,
                                              "end": 4094
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4090,
                                            "end": 4094
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3931,
                                        "end": 4116
                                      }
                                    },
                                    "loc": {
                                      "start": 3918,
                                      "end": 4116
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1784,
                                  "end": 4134
                                }
                              },
                              "loc": {
                                "start": 1768,
                                "end": 4134
                              }
                            }
                          ],
                          "loc": {
                            "start": 967,
                            "end": 4148
                          }
                        },
                        "loc": {
                          "start": 960,
                          "end": 4148
                        }
                      }
                    ],
                    "loc": {
                      "start": 914,
                      "end": 4158
                    }
                  },
                  "loc": {
                    "start": 909,
                    "end": 4158
                  }
                }
              ],
              "loc": {
                "start": 884,
                "end": 4164
              }
            },
            "loc": {
              "start": 878,
              "end": 4164
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 4169,
                "end": 4177
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursor",
                    "loc": {
                      "start": 4188,
                      "end": 4197
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4188,
                    "end": 4197
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 4206,
                      "end": 4217
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4206,
                    "end": 4217
                  }
                }
              ],
              "loc": {
                "start": 4178,
                "end": 4223
              }
            },
            "loc": {
              "start": 4169,
              "end": 4223
            }
          }
        ],
        "loc": {
          "start": 872,
          "end": 4227
        }
      },
      "loc": {
        "start": 839,
        "end": 4227
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 32,
          "end": 34
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 34
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 35,
          "end": 45
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 35,
        "end": 45
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 46,
          "end": 56
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 46,
        "end": 56
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 57,
          "end": 62
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 57,
        "end": 62
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 63,
          "end": 68
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 63,
        "end": 68
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 69,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 88,
                  "end": 92
                }
              },
              "loc": {
                "start": 88,
                "end": 92
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 106,
                      "end": 114
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 103,
                    "end": 114
                  }
                }
              ],
              "loc": {
                "start": 93,
                "end": 120
              }
            },
            "loc": {
              "start": 81,
              "end": 120
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 132,
                  "end": 136
                }
              },
              "loc": {
                "start": 132,
                "end": 136
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 150,
                      "end": 158
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 147,
                    "end": 158
                  }
                }
              ],
              "loc": {
                "start": 137,
                "end": 164
              }
            },
            "loc": {
              "start": 125,
              "end": 164
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 166
        }
      },
      "loc": {
        "start": 69,
        "end": 166
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 167,
          "end": 170
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 177,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 177,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 191,
                "end": 200
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 191,
              "end": 200
            }
          }
        ],
        "loc": {
          "start": 171,
          "end": 202
        }
      },
      "loc": {
        "start": 167,
        "end": 202
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 232,
          "end": 234
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 232,
        "end": 234
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 235,
          "end": 245
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 235,
        "end": 245
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 246,
          "end": 249
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 246,
        "end": 249
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 250,
          "end": 259
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 250,
        "end": 259
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 260,
          "end": 272
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 279,
                "end": 281
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 279,
              "end": 281
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 286,
                "end": 294
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 286,
              "end": 294
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 299,
                "end": 310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 299,
              "end": 310
            }
          }
        ],
        "loc": {
          "start": 273,
          "end": 312
        }
      },
      "loc": {
        "start": 260,
        "end": 312
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 313,
          "end": 316
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOwn",
              "loc": {
                "start": 323,
                "end": 328
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 323,
              "end": 328
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 333,
                "end": 345
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 333,
              "end": 345
            }
          }
        ],
        "loc": {
          "start": 317,
          "end": 347
        }
      },
      "loc": {
        "start": 313,
        "end": 347
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 378,
          "end": 380
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 378,
        "end": 380
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 381,
          "end": 392
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 381,
        "end": 392
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 393,
          "end": 399
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 393,
        "end": 399
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 400,
          "end": 412
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 400,
        "end": 412
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 413,
          "end": 416
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 423,
                "end": 436
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 423,
              "end": 436
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 441,
                "end": 450
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 441,
              "end": 450
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 455,
                "end": 466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 455,
              "end": 466
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 471,
                "end": 480
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 471,
              "end": 480
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 485,
                "end": 494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 485,
              "end": 494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 499,
                "end": 506
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 499,
              "end": 506
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 511,
                "end": 523
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 511,
              "end": 523
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 528,
                "end": 536
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 528,
              "end": 536
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 541,
                "end": 555
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 566,
                      "end": 568
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 566,
                    "end": 568
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 577,
                      "end": 587
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 577,
                    "end": 587
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 596,
                      "end": 606
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 596,
                    "end": 606
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 615,
                      "end": 622
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 615,
                    "end": 622
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 631,
                      "end": 642
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 631,
                    "end": 642
                  }
                }
              ],
              "loc": {
                "start": 556,
                "end": 648
              }
            },
            "loc": {
              "start": 541,
              "end": 648
            }
          }
        ],
        "loc": {
          "start": 417,
          "end": 650
        }
      },
      "loc": {
        "start": 413,
        "end": 650
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 681,
          "end": 683
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 681,
        "end": 683
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 684,
          "end": 694
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 684,
        "end": 694
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 695,
          "end": 705
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 695,
        "end": 705
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 706,
          "end": 717
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 706,
        "end": 717
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 718,
          "end": 724
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 718,
        "end": 724
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 725,
          "end": 730
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 725,
        "end": 730
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 731,
          "end": 751
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 731,
        "end": 751
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 752,
          "end": 756
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 752,
        "end": 756
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 757,
          "end": 769
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 757,
        "end": 769
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 10,
          "end": 20
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 24,
            "end": 29
          }
        },
        "loc": {
          "start": 24,
          "end": 29
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 32,
                "end": 34
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 34
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 35,
                "end": 45
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 35,
              "end": 45
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 46,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 46,
              "end": 56
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 57,
                "end": 62
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 57,
              "end": 62
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 63,
                "end": 68
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 68
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 69,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 88,
                        "end": 92
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 92
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 106,
                            "end": 114
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 103,
                          "end": 114
                        }
                      }
                    ],
                    "loc": {
                      "start": 93,
                      "end": 120
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 120
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 132,
                        "end": 136
                      }
                    },
                    "loc": {
                      "start": 132,
                      "end": 136
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 150,
                            "end": 158
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 147,
                          "end": 158
                        }
                      }
                    ],
                    "loc": {
                      "start": 137,
                      "end": 164
                    }
                  },
                  "loc": {
                    "start": 125,
                    "end": 164
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 166
              }
            },
            "loc": {
              "start": 69,
              "end": 166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 167,
                "end": 170
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 177,
                      "end": 186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 177,
                    "end": 186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 191,
                      "end": 200
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 191,
                    "end": 200
                  }
                }
              ],
              "loc": {
                "start": 171,
                "end": 202
              }
            },
            "loc": {
              "start": 167,
              "end": 202
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 204
        }
      },
      "loc": {
        "start": 1,
        "end": 204
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 214,
          "end": 222
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 226,
            "end": 229
          }
        },
        "loc": {
          "start": 226,
          "end": 229
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 232,
                "end": 234
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 232,
              "end": 234
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 235,
                "end": 245
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 235,
              "end": 245
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 246,
                "end": 249
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 246,
              "end": 249
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 250,
                "end": 259
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 250,
              "end": 259
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 260,
                "end": 272
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 279,
                      "end": 281
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 279,
                    "end": 281
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 286,
                      "end": 294
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 286,
                    "end": 294
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 299,
                      "end": 310
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 299,
                    "end": 310
                  }
                }
              ],
              "loc": {
                "start": 273,
                "end": 312
              }
            },
            "loc": {
              "start": 260,
              "end": 312
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 313,
                "end": 316
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 323,
                      "end": 328
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 323,
                    "end": 328
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 333,
                      "end": 345
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 333,
                    "end": 345
                  }
                }
              ],
              "loc": {
                "start": 317,
                "end": 347
              }
            },
            "loc": {
              "start": 313,
              "end": 347
            }
          }
        ],
        "loc": {
          "start": 230,
          "end": 349
        }
      },
      "loc": {
        "start": 205,
        "end": 349
      }
    },
    "Team_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_nav",
        "loc": {
          "start": 359,
          "end": 367
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 371,
            "end": 375
          }
        },
        "loc": {
          "start": 371,
          "end": 375
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 378,
                "end": 380
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 378,
              "end": 380
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 381,
                "end": 392
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 381,
              "end": 392
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 393,
                "end": 399
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 393,
              "end": 399
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 400,
                "end": 412
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 400,
              "end": 412
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 413,
                "end": 416
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 423,
                      "end": 436
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 423,
                    "end": 436
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 441,
                      "end": 450
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 441,
                    "end": 450
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 455,
                      "end": 466
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 455,
                    "end": 466
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 471,
                      "end": 480
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 471,
                    "end": 480
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 485,
                      "end": 494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 485,
                    "end": 494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 499,
                      "end": 506
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 499,
                    "end": 506
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 511,
                      "end": 523
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 511,
                    "end": 523
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 528,
                      "end": 536
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 528,
                    "end": 536
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 541,
                      "end": 555
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 566,
                            "end": 568
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 566,
                          "end": 568
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 577,
                            "end": 587
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 577,
                          "end": 587
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 596,
                            "end": 606
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 596,
                          "end": 606
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 615,
                            "end": 622
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 615,
                          "end": 622
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 631,
                            "end": 642
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 631,
                          "end": 642
                        }
                      }
                    ],
                    "loc": {
                      "start": 556,
                      "end": 648
                    }
                  },
                  "loc": {
                    "start": 541,
                    "end": 648
                  }
                }
              ],
              "loc": {
                "start": 417,
                "end": 650
              }
            },
            "loc": {
              "start": 413,
              "end": 650
            }
          }
        ],
        "loc": {
          "start": 376,
          "end": 652
        }
      },
      "loc": {
        "start": 350,
        "end": 652
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 662,
          "end": 670
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 674,
            "end": 678
          }
        },
        "loc": {
          "start": 674,
          "end": 678
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 681,
                "end": 683
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 681,
              "end": 683
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 684,
                "end": 694
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 684,
              "end": 694
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 695,
                "end": 705
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 695,
              "end": 705
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 706,
                "end": 717
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 706,
              "end": 717
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 718,
                "end": 724
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 718,
              "end": 724
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 725,
                "end": 730
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 725,
              "end": 730
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 731,
                "end": 751
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 731,
              "end": 751
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 752,
                "end": 756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 752,
              "end": 756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 757,
                "end": 769
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 757,
              "end": 769
            }
          }
        ],
        "loc": {
          "start": 679,
          "end": 771
        }
      },
      "loc": {
        "start": 653,
        "end": 771
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "runRoutineOutputs",
      "loc": {
        "start": 779,
        "end": 796
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 798,
              "end": 803
            }
          },
          "loc": {
            "start": 797,
            "end": 803
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "RunRoutineOutputSearchInput",
              "loc": {
                "start": 805,
                "end": 832
              }
            },
            "loc": {
              "start": 805,
              "end": 832
            }
          },
          "loc": {
            "start": 805,
            "end": 833
          }
        },
        "directives": [],
        "loc": {
          "start": 797,
          "end": 833
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "runRoutineOutputs",
            "loc": {
              "start": 839,
              "end": 856
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 857,
                  "end": 862
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 865,
                    "end": 870
                  }
                },
                "loc": {
                  "start": 864,
                  "end": 870
                }
              },
              "loc": {
                "start": 857,
                "end": 870
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "edges",
                  "loc": {
                    "start": 878,
                    "end": 883
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "cursor",
                        "loc": {
                          "start": 894,
                          "end": 900
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 894,
                        "end": 900
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 909,
                          "end": 913
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 928,
                                "end": 930
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 928,
                              "end": 930
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "data",
                              "loc": {
                                "start": 943,
                                "end": 947
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 943,
                              "end": 947
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "output",
                              "loc": {
                                "start": 960,
                                "end": 966
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 985,
                                      "end": 987
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 985,
                                    "end": 987
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "index",
                                    "loc": {
                                      "start": 1004,
                                      "end": 1009
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 1004,
                                    "end": 1009
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "name",
                                    "loc": {
                                      "start": 1026,
                                      "end": 1030
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 1026,
                                    "end": 1030
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "routineVersion",
                                    "loc": {
                                      "start": 1047,
                                      "end": 1061
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 1084,
                                            "end": 1086
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1084,
                                          "end": 1086
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "complexity",
                                          "loc": {
                                            "start": 1107,
                                            "end": 1117
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1107,
                                          "end": 1117
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isAutomatable",
                                          "loc": {
                                            "start": 1138,
                                            "end": 1151
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1138,
                                          "end": 1151
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isComplete",
                                          "loc": {
                                            "start": 1172,
                                            "end": 1182
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1172,
                                          "end": 1182
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isDeleted",
                                          "loc": {
                                            "start": 1203,
                                            "end": 1212
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1203,
                                          "end": 1212
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isLatest",
                                          "loc": {
                                            "start": 1233,
                                            "end": 1241
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1233,
                                          "end": 1241
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isPrivate",
                                          "loc": {
                                            "start": 1262,
                                            "end": 1271
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1262,
                                          "end": 1271
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "root",
                                          "loc": {
                                            "start": 1292,
                                            "end": 1296
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 1323,
                                                  "end": 1325
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1323,
                                                "end": 1325
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isInternal",
                                                "loc": {
                                                  "start": 1350,
                                                  "end": 1360
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1350,
                                                "end": 1360
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isPrivate",
                                                "loc": {
                                                  "start": 1385,
                                                  "end": 1394
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1385,
                                                "end": 1394
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1297,
                                            "end": 1416
                                          }
                                        },
                                        "loc": {
                                          "start": 1292,
                                          "end": 1416
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "routineType",
                                          "loc": {
                                            "start": 1437,
                                            "end": 1448
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1437,
                                          "end": 1448
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 1469,
                                            "end": 1481
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 1508,
                                                  "end": 1510
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1508,
                                                "end": 1510
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 1535,
                                                  "end": 1543
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1535,
                                                "end": 1543
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 1568,
                                                  "end": 1579
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1568,
                                                "end": 1579
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "instructions",
                                                "loc": {
                                                  "start": 1604,
                                                  "end": 1616
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1604,
                                                "end": 1616
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 1641,
                                                  "end": 1645
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1641,
                                                "end": 1645
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1482,
                                            "end": 1667
                                          }
                                        },
                                        "loc": {
                                          "start": 1469,
                                          "end": 1667
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "versionIndex",
                                          "loc": {
                                            "start": 1688,
                                            "end": 1700
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1688,
                                          "end": 1700
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "versionLabel",
                                          "loc": {
                                            "start": 1721,
                                            "end": 1733
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1721,
                                          "end": 1733
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 1062,
                                      "end": 1751
                                    }
                                  },
                                  "loc": {
                                    "start": 1047,
                                    "end": 1751
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "standardVersion",
                                    "loc": {
                                      "start": 1768,
                                      "end": 1783
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 1806,
                                            "end": 1808
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1806,
                                          "end": 1808
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 1829,
                                            "end": 1839
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1829,
                                          "end": 1839
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 1860,
                                            "end": 1870
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1860,
                                          "end": 1870
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isComplete",
                                          "loc": {
                                            "start": 1891,
                                            "end": 1901
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1891,
                                          "end": 1901
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isFile",
                                          "loc": {
                                            "start": 1922,
                                            "end": 1928
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1922,
                                          "end": 1928
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isLatest",
                                          "loc": {
                                            "start": 1949,
                                            "end": 1957
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1949,
                                          "end": 1957
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isPrivate",
                                          "loc": {
                                            "start": 1978,
                                            "end": 1987
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1978,
                                          "end": 1987
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "default",
                                          "loc": {
                                            "start": 2008,
                                            "end": 2015
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2008,
                                          "end": 2015
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "standardType",
                                          "loc": {
                                            "start": 2036,
                                            "end": 2048
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2036,
                                          "end": 2048
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "props",
                                          "loc": {
                                            "start": 2069,
                                            "end": 2074
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2069,
                                          "end": 2074
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "yup",
                                          "loc": {
                                            "start": 2095,
                                            "end": 2098
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2095,
                                          "end": 2098
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "versionIndex",
                                          "loc": {
                                            "start": 2119,
                                            "end": 2131
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2119,
                                          "end": 2131
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "versionLabel",
                                          "loc": {
                                            "start": 2152,
                                            "end": 2164
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2152,
                                          "end": 2164
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "commentsCount",
                                          "loc": {
                                            "start": 2185,
                                            "end": 2198
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2185,
                                          "end": 2198
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "directoryListingsCount",
                                          "loc": {
                                            "start": 2219,
                                            "end": 2241
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2219,
                                          "end": 2241
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "forksCount",
                                          "loc": {
                                            "start": 2262,
                                            "end": 2272
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2262,
                                          "end": 2272
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reportsCount",
                                          "loc": {
                                            "start": 2293,
                                            "end": 2305
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2293,
                                          "end": 2305
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "you",
                                          "loc": {
                                            "start": 2326,
                                            "end": 2329
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "canComment",
                                                "loc": {
                                                  "start": 2356,
                                                  "end": 2366
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2356,
                                                "end": 2366
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "canCopy",
                                                "loc": {
                                                  "start": 2391,
                                                  "end": 2398
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2391,
                                                "end": 2398
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "canDelete",
                                                "loc": {
                                                  "start": 2423,
                                                  "end": 2432
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2423,
                                                "end": 2432
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "canReport",
                                                "loc": {
                                                  "start": 2457,
                                                  "end": 2466
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2457,
                                                "end": 2466
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "canUpdate",
                                                "loc": {
                                                  "start": 2491,
                                                  "end": 2500
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2491,
                                                "end": 2500
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "canUse",
                                                "loc": {
                                                  "start": 2525,
                                                  "end": 2531
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2525,
                                                "end": 2531
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "canRead",
                                                "loc": {
                                                  "start": 2556,
                                                  "end": 2563
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2556,
                                                "end": 2563
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 2330,
                                            "end": 2585
                                          }
                                        },
                                        "loc": {
                                          "start": 2326,
                                          "end": 2585
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "root",
                                          "loc": {
                                            "start": 2606,
                                            "end": 2610
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 2637,
                                                  "end": 2639
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2637,
                                                "end": 2639
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 2664,
                                                  "end": 2674
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2664,
                                                "end": 2674
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 2699,
                                                  "end": 2709
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2699,
                                                "end": 2709
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isPrivate",
                                                "loc": {
                                                  "start": 2734,
                                                  "end": 2743
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2734,
                                                "end": 2743
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "issuesCount",
                                                "loc": {
                                                  "start": 2768,
                                                  "end": 2779
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2768,
                                                "end": 2779
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "labels",
                                                "loc": {
                                                  "start": 2804,
                                                  "end": 2810
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "FragmentSpread",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "Label_list",
                                                      "loc": {
                                                        "start": 2844,
                                                        "end": 2854
                                                      }
                                                    },
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2841,
                                                      "end": 2854
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 2811,
                                                  "end": 2880
                                                }
                                              },
                                              "loc": {
                                                "start": 2804,
                                                "end": 2880
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "owner",
                                                "loc": {
                                                  "start": 2905,
                                                  "end": 2910
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "InlineFragment",
                                                    "typeCondition": {
                                                      "kind": "NamedType",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "Team",
                                                        "loc": {
                                                          "start": 2948,
                                                          "end": 2952
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 2948,
                                                        "end": 2952
                                                      }
                                                    },
                                                    "directives": [],
                                                    "selectionSet": {
                                                      "kind": "SelectionSet",
                                                      "selections": [
                                                        {
                                                          "kind": "FragmentSpread",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "Team_nav",
                                                            "loc": {
                                                              "start": 2990,
                                                              "end": 2998
                                                            }
                                                          },
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2987,
                                                            "end": 2998
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 2953,
                                                        "end": 3028
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 2941,
                                                      "end": 3028
                                                    }
                                                  },
                                                  {
                                                    "kind": "InlineFragment",
                                                    "typeCondition": {
                                                      "kind": "NamedType",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "User",
                                                        "loc": {
                                                          "start": 3064,
                                                          "end": 3068
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 3064,
                                                        "end": 3068
                                                      }
                                                    },
                                                    "directives": [],
                                                    "selectionSet": {
                                                      "kind": "SelectionSet",
                                                      "selections": [
                                                        {
                                                          "kind": "FragmentSpread",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "User_nav",
                                                            "loc": {
                                                              "start": 3106,
                                                              "end": 3114
                                                            }
                                                          },
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 3103,
                                                            "end": 3114
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 3069,
                                                        "end": 3144
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 3057,
                                                      "end": 3144
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 2911,
                                                  "end": 3170
                                                }
                                              },
                                              "loc": {
                                                "start": 2905,
                                                "end": 3170
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "permissions",
                                                "loc": {
                                                  "start": 3195,
                                                  "end": 3206
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3195,
                                                "end": 3206
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "questionsCount",
                                                "loc": {
                                                  "start": 3231,
                                                  "end": 3245
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3231,
                                                "end": 3245
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "score",
                                                "loc": {
                                                  "start": 3270,
                                                  "end": 3275
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3270,
                                                "end": 3275
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "bookmarks",
                                                "loc": {
                                                  "start": 3300,
                                                  "end": 3309
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3300,
                                                "end": 3309
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "tags",
                                                "loc": {
                                                  "start": 3334,
                                                  "end": 3338
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "FragmentSpread",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "Tag_list",
                                                      "loc": {
                                                        "start": 3372,
                                                        "end": 3380
                                                      }
                                                    },
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3369,
                                                      "end": 3380
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3339,
                                                  "end": 3406
                                                }
                                              },
                                              "loc": {
                                                "start": 3334,
                                                "end": 3406
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "transfersCount",
                                                "loc": {
                                                  "start": 3431,
                                                  "end": 3445
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3431,
                                                "end": 3445
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "views",
                                                "loc": {
                                                  "start": 3470,
                                                  "end": 3475
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3470,
                                                "end": 3475
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "you",
                                                "loc": {
                                                  "start": 3500,
                                                  "end": 3503
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "canDelete",
                                                      "loc": {
                                                        "start": 3534,
                                                        "end": 3543
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3534,
                                                      "end": 3543
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "canBookmark",
                                                      "loc": {
                                                        "start": 3572,
                                                        "end": 3583
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3572,
                                                      "end": 3583
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "canTransfer",
                                                      "loc": {
                                                        "start": 3612,
                                                        "end": 3623
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3612,
                                                      "end": 3623
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "canUpdate",
                                                      "loc": {
                                                        "start": 3652,
                                                        "end": 3661
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3652,
                                                      "end": 3661
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "canRead",
                                                      "loc": {
                                                        "start": 3690,
                                                        "end": 3697
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3690,
                                                      "end": 3697
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "canReact",
                                                      "loc": {
                                                        "start": 3726,
                                                        "end": 3734
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3726,
                                                      "end": 3734
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isBookmarked",
                                                      "loc": {
                                                        "start": 3763,
                                                        "end": 3775
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3763,
                                                      "end": 3775
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isViewed",
                                                      "loc": {
                                                        "start": 3804,
                                                        "end": 3812
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3804,
                                                      "end": 3812
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "reaction",
                                                      "loc": {
                                                        "start": 3841,
                                                        "end": 3849
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3841,
                                                      "end": 3849
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3504,
                                                  "end": 3875
                                                }
                                              },
                                              "loc": {
                                                "start": 3500,
                                                "end": 3875
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 2611,
                                            "end": 3897
                                          }
                                        },
                                        "loc": {
                                          "start": 2606,
                                          "end": 3897
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 3918,
                                            "end": 3930
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 3957,
                                                  "end": 3959
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3957,
                                                "end": 3959
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 3984,
                                                  "end": 3992
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3984,
                                                "end": 3992
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 4017,
                                                  "end": 4028
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4017,
                                                "end": 4028
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "jsonVariable",
                                                "loc": {
                                                  "start": 4053,
                                                  "end": 4065
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4053,
                                                "end": 4065
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 4090,
                                                  "end": 4094
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4090,
                                                "end": 4094
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3931,
                                            "end": 4116
                                          }
                                        },
                                        "loc": {
                                          "start": 3918,
                                          "end": 4116
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 1784,
                                      "end": 4134
                                    }
                                  },
                                  "loc": {
                                    "start": 1768,
                                    "end": 4134
                                  }
                                }
                              ],
                              "loc": {
                                "start": 967,
                                "end": 4148
                              }
                            },
                            "loc": {
                              "start": 960,
                              "end": 4148
                            }
                          }
                        ],
                        "loc": {
                          "start": 914,
                          "end": 4158
                        }
                      },
                      "loc": {
                        "start": 909,
                        "end": 4158
                      }
                    }
                  ],
                  "loc": {
                    "start": 884,
                    "end": 4164
                  }
                },
                "loc": {
                  "start": 878,
                  "end": 4164
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 4169,
                    "end": 4177
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursor",
                        "loc": {
                          "start": 4188,
                          "end": 4197
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 4188,
                        "end": 4197
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 4206,
                          "end": 4217
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 4206,
                        "end": 4217
                      }
                    }
                  ],
                  "loc": {
                    "start": 4178,
                    "end": 4223
                  }
                },
                "loc": {
                  "start": 4169,
                  "end": 4223
                }
              }
            ],
            "loc": {
              "start": 872,
              "end": 4227
            }
          },
          "loc": {
            "start": 839,
            "end": 4227
          }
        }
      ],
      "loc": {
        "start": 835,
        "end": 4229
      }
    },
    "loc": {
      "start": 773,
      "end": 4229
    }
  },
  "variableValues": {},
  "path": {
    "key": "runRoutineOutput_findMany"
  }
} as const;
