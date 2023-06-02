export const auth_switchCurrentAccount = {
  "fieldName": "switchCurrentAccount",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "switchCurrentAccount",
        "loc": {
          "start": 346,
          "end": 366
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 367,
              "end": 372
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 375,
                "end": 380
              }
            },
            "loc": {
              "start": 374,
              "end": 380
            }
          },
          "loc": {
            "start": 367,
            "end": 380
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
              "value": "isLoggedIn",
              "loc": {
                "start": 388,
                "end": 398
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 388,
              "end": 398
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 403,
                "end": 411
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 403,
              "end": 411
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 416,
                "end": 421
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
                    "value": "activeFocusMode",
                    "loc": {
                      "start": 432,
                      "end": 447
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
                          "value": "mode",
                          "loc": {
                            "start": 462,
                            "end": 466
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
                                "value": "filters",
                                "loc": {
                                  "start": 485,
                                  "end": 492
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
                                        "start": 515,
                                        "end": 517
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 515,
                                      "end": 517
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "filterType",
                                      "loc": {
                                        "start": 538,
                                        "end": 548
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 538,
                                      "end": 548
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 569,
                                        "end": 572
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
                                              "start": 599,
                                              "end": 601
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 599,
                                            "end": 601
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 626,
                                              "end": 636
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 626,
                                            "end": 636
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tag",
                                            "loc": {
                                              "start": 661,
                                              "end": 664
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 661,
                                            "end": 664
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 689,
                                              "end": 698
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 689,
                                            "end": 698
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 723,
                                              "end": 735
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
                                                    "start": 766,
                                                    "end": 768
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 766,
                                                  "end": 768
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 797,
                                                    "end": 805
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 797,
                                                  "end": 805
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 834,
                                                    "end": 845
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 834,
                                                  "end": 845
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 736,
                                              "end": 871
                                            }
                                          },
                                          "loc": {
                                            "start": 723,
                                            "end": 871
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 896,
                                              "end": 899
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
                                                    "start": 930,
                                                    "end": 935
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 930,
                                                  "end": 935
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 964,
                                                    "end": 976
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 964,
                                                  "end": 976
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 900,
                                              "end": 1002
                                            }
                                          },
                                          "loc": {
                                            "start": 896,
                                            "end": 1002
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 573,
                                        "end": 1024
                                      }
                                    },
                                    "loc": {
                                      "start": 569,
                                      "end": 1024
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "focusMode",
                                      "loc": {
                                        "start": 1045,
                                        "end": 1054
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
                                            "value": "labels",
                                            "loc": {
                                              "start": 1081,
                                              "end": 1087
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
                                                    "start": 1118,
                                                    "end": 1120
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1118,
                                                  "end": 1120
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "color",
                                                  "loc": {
                                                    "start": 1149,
                                                    "end": 1154
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1149,
                                                  "end": 1154
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "label",
                                                  "loc": {
                                                    "start": 1183,
                                                    "end": 1188
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1183,
                                                  "end": 1188
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1088,
                                              "end": 1214
                                            }
                                          },
                                          "loc": {
                                            "start": 1081,
                                            "end": 1214
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 1239,
                                              "end": 1247
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
                                                  "value": "Schedule_common",
                                                  "loc": {
                                                    "start": 1281,
                                                    "end": 1296
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 1278,
                                                  "end": 1296
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1248,
                                              "end": 1322
                                            }
                                          },
                                          "loc": {
                                            "start": 1239,
                                            "end": 1322
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1347,
                                              "end": 1349
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1347,
                                            "end": 1349
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1374,
                                              "end": 1378
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1374,
                                            "end": 1378
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1403,
                                              "end": 1414
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1403,
                                            "end": 1414
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1055,
                                        "end": 1436
                                      }
                                    },
                                    "loc": {
                                      "start": 1045,
                                      "end": 1436
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 493,
                                  "end": 1454
                                }
                              },
                              "loc": {
                                "start": 485,
                                "end": 1454
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 1471,
                                  "end": 1477
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
                                        "start": 1500,
                                        "end": 1502
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1500,
                                      "end": 1502
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 1523,
                                        "end": 1528
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1523,
                                      "end": 1528
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 1549,
                                        "end": 1554
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1549,
                                      "end": 1554
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1478,
                                  "end": 1572
                                }
                              },
                              "loc": {
                                "start": 1471,
                                "end": 1572
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 1589,
                                  "end": 1601
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
                                        "start": 1624,
                                        "end": 1626
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1624,
                                      "end": 1626
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 1647,
                                        "end": 1657
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1647,
                                      "end": 1657
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 1678,
                                        "end": 1688
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1678,
                                      "end": 1688
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 1709,
                                        "end": 1718
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
                                              "start": 1745,
                                              "end": 1747
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1745,
                                            "end": 1747
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 1772,
                                              "end": 1782
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1772,
                                            "end": 1782
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 1807,
                                              "end": 1817
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1807,
                                            "end": 1817
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1842,
                                              "end": 1846
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1842,
                                            "end": 1846
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1871,
                                              "end": 1882
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1871,
                                            "end": 1882
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 1907,
                                              "end": 1914
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1907,
                                            "end": 1914
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 1939,
                                              "end": 1944
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1939,
                                            "end": 1944
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 1969,
                                              "end": 1979
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1969,
                                            "end": 1979
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 2004,
                                              "end": 2017
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
                                                    "start": 2048,
                                                    "end": 2050
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2048,
                                                  "end": 2050
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 2079,
                                                    "end": 2089
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2079,
                                                  "end": 2089
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 2118,
                                                    "end": 2128
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2118,
                                                  "end": 2128
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 2157,
                                                    "end": 2161
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2157,
                                                  "end": 2161
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 2190,
                                                    "end": 2201
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2190,
                                                  "end": 2201
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 2230,
                                                    "end": 2237
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2230,
                                                  "end": 2237
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 2266,
                                                    "end": 2271
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2266,
                                                  "end": 2271
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 2300,
                                                    "end": 2310
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2300,
                                                  "end": 2310
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2018,
                                              "end": 2336
                                            }
                                          },
                                          "loc": {
                                            "start": 2004,
                                            "end": 2336
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1719,
                                        "end": 2358
                                      }
                                    },
                                    "loc": {
                                      "start": 1709,
                                      "end": 2358
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1602,
                                  "end": 2376
                                }
                              },
                              "loc": {
                                "start": 1589,
                                "end": 2376
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 2393,
                                  "end": 2405
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
                                        "start": 2428,
                                        "end": 2430
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2428,
                                      "end": 2430
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2451,
                                        "end": 2461
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2451,
                                      "end": 2461
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 2482,
                                        "end": 2494
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
                                              "start": 2521,
                                              "end": 2523
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2521,
                                            "end": 2523
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 2548,
                                              "end": 2556
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2548,
                                            "end": 2556
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 2581,
                                              "end": 2592
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2581,
                                            "end": 2592
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 2617,
                                              "end": 2621
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2617,
                                            "end": 2621
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2495,
                                        "end": 2643
                                      }
                                    },
                                    "loc": {
                                      "start": 2482,
                                      "end": 2643
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 2664,
                                        "end": 2673
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
                                              "start": 2700,
                                              "end": 2702
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2700,
                                            "end": 2702
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 2727,
                                              "end": 2732
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2727,
                                            "end": 2732
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 2757,
                                              "end": 2761
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2757,
                                            "end": 2761
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 2786,
                                              "end": 2793
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2786,
                                            "end": 2793
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 2818,
                                              "end": 2830
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
                                                    "start": 2861,
                                                    "end": 2863
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2861,
                                                  "end": 2863
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 2892,
                                                    "end": 2900
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2892,
                                                  "end": 2900
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 2929,
                                                    "end": 2940
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2929,
                                                  "end": 2940
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 2969,
                                                    "end": 2973
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2969,
                                                  "end": 2973
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2831,
                                              "end": 2999
                                            }
                                          },
                                          "loc": {
                                            "start": 2818,
                                            "end": 2999
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2674,
                                        "end": 3021
                                      }
                                    },
                                    "loc": {
                                      "start": 2664,
                                      "end": 3021
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2406,
                                  "end": 3039
                                }
                              },
                              "loc": {
                                "start": 2393,
                                "end": 3039
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 3056,
                                  "end": 3064
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
                                      "value": "Schedule_common",
                                      "loc": {
                                        "start": 3090,
                                        "end": 3105
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 3087,
                                      "end": 3105
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3065,
                                  "end": 3123
                                }
                              },
                              "loc": {
                                "start": 3056,
                                "end": 3123
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3140,
                                  "end": 3142
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3140,
                                "end": 3142
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3159,
                                  "end": 3163
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3159,
                                "end": 3163
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 3180,
                                  "end": 3191
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3180,
                                "end": 3191
                              }
                            }
                          ],
                          "loc": {
                            "start": 467,
                            "end": 3205
                          }
                        },
                        "loc": {
                          "start": 462,
                          "end": 3205
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 3218,
                            "end": 3231
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3218,
                          "end": 3231
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 3244,
                            "end": 3252
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3244,
                          "end": 3252
                        }
                      }
                    ],
                    "loc": {
                      "start": 448,
                      "end": 3262
                    }
                  },
                  "loc": {
                    "start": 432,
                    "end": 3262
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 3271,
                      "end": 3280
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3271,
                    "end": 3280
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 3289,
                      "end": 3302
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
                            "start": 3317,
                            "end": 3319
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3317,
                          "end": 3319
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3332,
                            "end": 3342
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3332,
                          "end": 3342
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3355,
                            "end": 3365
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3355,
                          "end": 3365
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 3378,
                            "end": 3383
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3378,
                          "end": 3383
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 3396,
                            "end": 3410
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3396,
                          "end": 3410
                        }
                      }
                    ],
                    "loc": {
                      "start": 3303,
                      "end": 3420
                    }
                  },
                  "loc": {
                    "start": 3289,
                    "end": 3420
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 3429,
                      "end": 3439
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
                          "value": "filters",
                          "loc": {
                            "start": 3454,
                            "end": 3461
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
                                  "start": 3480,
                                  "end": 3482
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3480,
                                "end": 3482
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 3499,
                                  "end": 3509
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3499,
                                "end": 3509
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 3526,
                                  "end": 3529
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
                                        "start": 3552,
                                        "end": 3554
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3552,
                                      "end": 3554
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3575,
                                        "end": 3585
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3575,
                                      "end": 3585
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 3606,
                                        "end": 3609
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3606,
                                      "end": 3609
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 3630,
                                        "end": 3639
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3630,
                                      "end": 3639
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 3660,
                                        "end": 3672
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
                                              "start": 3699,
                                              "end": 3701
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3699,
                                            "end": 3701
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
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
                                            "value": "description",
                                            "loc": {
                                              "start": 3759,
                                              "end": 3770
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3759,
                                            "end": 3770
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3673,
                                        "end": 3792
                                      }
                                    },
                                    "loc": {
                                      "start": 3660,
                                      "end": 3792
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 3813,
                                        "end": 3816
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
                                              "start": 3843,
                                              "end": 3848
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3843,
                                            "end": 3848
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 3873,
                                              "end": 3885
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3873,
                                            "end": 3885
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3817,
                                        "end": 3907
                                      }
                                    },
                                    "loc": {
                                      "start": 3813,
                                      "end": 3907
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3530,
                                  "end": 3925
                                }
                              },
                              "loc": {
                                "start": 3526,
                                "end": 3925
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 3942,
                                  "end": 3951
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
                                      "value": "labels",
                                      "loc": {
                                        "start": 3974,
                                        "end": 3980
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
                                              "start": 4007,
                                              "end": 4009
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4007,
                                            "end": 4009
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 4034,
                                              "end": 4039
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4034,
                                            "end": 4039
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 4064,
                                              "end": 4069
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4064,
                                            "end": 4069
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3981,
                                        "end": 4091
                                      }
                                    },
                                    "loc": {
                                      "start": 3974,
                                      "end": 4091
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 4112,
                                        "end": 4120
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
                                            "value": "Schedule_common",
                                            "loc": {
                                              "start": 4150,
                                              "end": 4165
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 4147,
                                            "end": 4165
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4121,
                                        "end": 4187
                                      }
                                    },
                                    "loc": {
                                      "start": 4112,
                                      "end": 4187
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4208,
                                        "end": 4210
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4208,
                                      "end": 4210
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4231,
                                        "end": 4235
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4231,
                                      "end": 4235
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4256,
                                        "end": 4267
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4256,
                                      "end": 4267
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3952,
                                  "end": 4285
                                }
                              },
                              "loc": {
                                "start": 3942,
                                "end": 4285
                              }
                            }
                          ],
                          "loc": {
                            "start": 3462,
                            "end": 4299
                          }
                        },
                        "loc": {
                          "start": 3454,
                          "end": 4299
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 4312,
                            "end": 4318
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
                                  "start": 4337,
                                  "end": 4339
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4337,
                                "end": 4339
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 4356,
                                  "end": 4361
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4356,
                                "end": 4361
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 4378,
                                  "end": 4383
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4378,
                                "end": 4383
                              }
                            }
                          ],
                          "loc": {
                            "start": 4319,
                            "end": 4397
                          }
                        },
                        "loc": {
                          "start": 4312,
                          "end": 4397
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 4410,
                            "end": 4422
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
                                  "start": 4441,
                                  "end": 4443
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4441,
                                "end": 4443
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4460,
                                  "end": 4470
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4460,
                                "end": 4470
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4487,
                                  "end": 4497
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4487,
                                "end": 4497
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 4514,
                                  "end": 4523
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
                                        "start": 4546,
                                        "end": 4548
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4546,
                                      "end": 4548
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4569,
                                        "end": 4579
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4569,
                                      "end": 4579
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4600,
                                        "end": 4610
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4600,
                                      "end": 4610
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4631,
                                        "end": 4635
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4631,
                                      "end": 4635
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4656,
                                        "end": 4667
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4656,
                                      "end": 4667
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 4688,
                                        "end": 4695
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4688,
                                      "end": 4695
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 4716,
                                        "end": 4721
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4716,
                                      "end": 4721
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 4742,
                                        "end": 4752
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4742,
                                      "end": 4752
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 4773,
                                        "end": 4786
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
                                              "start": 4813,
                                              "end": 4815
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4813,
                                            "end": 4815
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 4840,
                                              "end": 4850
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4840,
                                            "end": 4850
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 4875,
                                              "end": 4885
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4875,
                                            "end": 4885
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4910,
                                              "end": 4914
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4910,
                                            "end": 4914
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4939,
                                              "end": 4950
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4939,
                                            "end": 4950
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 4975,
                                              "end": 4982
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4975,
                                            "end": 4982
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 5007,
                                              "end": 5012
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5007,
                                            "end": 5012
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 5037,
                                              "end": 5047
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5037,
                                            "end": 5047
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4787,
                                        "end": 5069
                                      }
                                    },
                                    "loc": {
                                      "start": 4773,
                                      "end": 5069
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4524,
                                  "end": 5087
                                }
                              },
                              "loc": {
                                "start": 4514,
                                "end": 5087
                              }
                            }
                          ],
                          "loc": {
                            "start": 4423,
                            "end": 5101
                          }
                        },
                        "loc": {
                          "start": 4410,
                          "end": 5101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 5114,
                            "end": 5126
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
                                  "start": 5145,
                                  "end": 5147
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5145,
                                "end": 5147
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 5164,
                                  "end": 5174
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5164,
                                "end": 5174
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 5191,
                                  "end": 5203
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
                                        "start": 5226,
                                        "end": 5228
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5226,
                                      "end": 5228
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 5249,
                                        "end": 5257
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5249,
                                      "end": 5257
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 5278,
                                        "end": 5289
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5278,
                                      "end": 5289
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 5310,
                                        "end": 5314
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5310,
                                      "end": 5314
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5204,
                                  "end": 5332
                                }
                              },
                              "loc": {
                                "start": 5191,
                                "end": 5332
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 5349,
                                  "end": 5358
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
                                        "start": 5381,
                                        "end": 5383
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5381,
                                      "end": 5383
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 5404,
                                        "end": 5409
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5404,
                                      "end": 5409
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 5430,
                                        "end": 5434
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5430,
                                      "end": 5434
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 5455,
                                        "end": 5462
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5455,
                                      "end": 5462
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5483,
                                        "end": 5495
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
                                              "start": 5522,
                                              "end": 5524
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5522,
                                            "end": 5524
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5549,
                                              "end": 5557
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5549,
                                            "end": 5557
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5582,
                                              "end": 5593
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5582,
                                            "end": 5593
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 5618,
                                              "end": 5622
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5618,
                                            "end": 5622
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5496,
                                        "end": 5644
                                      }
                                    },
                                    "loc": {
                                      "start": 5483,
                                      "end": 5644
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5359,
                                  "end": 5662
                                }
                              },
                              "loc": {
                                "start": 5349,
                                "end": 5662
                              }
                            }
                          ],
                          "loc": {
                            "start": 5127,
                            "end": 5676
                          }
                        },
                        "loc": {
                          "start": 5114,
                          "end": 5676
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 5689,
                            "end": 5697
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
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 5719,
                                  "end": 5734
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 5716,
                                "end": 5734
                              }
                            }
                          ],
                          "loc": {
                            "start": 5698,
                            "end": 5748
                          }
                        },
                        "loc": {
                          "start": 5689,
                          "end": 5748
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5761,
                            "end": 5763
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5761,
                          "end": 5763
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5776,
                            "end": 5780
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5776,
                          "end": 5780
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5793,
                            "end": 5804
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5793,
                          "end": 5804
                        }
                      }
                    ],
                    "loc": {
                      "start": 3440,
                      "end": 5814
                    }
                  },
                  "loc": {
                    "start": 3429,
                    "end": 5814
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 5823,
                      "end": 5829
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5823,
                    "end": 5829
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 5838,
                      "end": 5848
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5838,
                    "end": 5848
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5857,
                      "end": 5859
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5857,
                    "end": 5859
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 5868,
                      "end": 5877
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5868,
                    "end": 5877
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 5886,
                      "end": 5902
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5886,
                    "end": 5902
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5911,
                      "end": 5915
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5911,
                    "end": 5915
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 5924,
                      "end": 5934
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5924,
                    "end": 5934
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 5943,
                      "end": 5956
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5943,
                    "end": 5956
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 5965,
                      "end": 5984
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5965,
                    "end": 5984
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 5993,
                      "end": 6006
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5993,
                    "end": 6006
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractsCount",
                    "loc": {
                      "start": 6015,
                      "end": 6034
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6015,
                    "end": 6034
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 6043,
                      "end": 6057
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6043,
                    "end": 6057
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 6066,
                      "end": 6071
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6066,
                    "end": 6071
                  }
                }
              ],
              "loc": {
                "start": 422,
                "end": 6077
              }
            },
            "loc": {
              "start": 416,
              "end": 6077
            }
          }
        ],
        "loc": {
          "start": 382,
          "end": 6081
        }
      },
      "loc": {
        "start": 346,
        "end": 6081
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 9,
          "end": 24
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 28,
            "end": 36
          }
        },
        "loc": {
          "start": 28,
          "end": 36
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
                "start": 39,
                "end": 41
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 39,
              "end": 41
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 42,
                "end": 52
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 42,
              "end": 52
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 53,
                "end": 63
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 53,
              "end": 63
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 64,
                "end": 73
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 64,
              "end": 73
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 74,
                "end": 81
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 74,
              "end": 81
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 82,
                "end": 90
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 82,
              "end": 90
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 91,
                "end": 101
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
                      "start": 108,
                      "end": 110
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 108,
                    "end": 110
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 115,
                      "end": 132
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 115,
                    "end": 132
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 137,
                      "end": 149
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 137,
                    "end": 149
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 154,
                      "end": 164
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 154,
                    "end": 164
                  }
                }
              ],
              "loc": {
                "start": 102,
                "end": 166
              }
            },
            "loc": {
              "start": 91,
              "end": 166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 167,
                "end": 178
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
                      "start": 185,
                      "end": 187
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 185,
                    "end": 187
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 192,
                      "end": 206
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 192,
                    "end": 206
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 211,
                      "end": 219
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 211,
                    "end": 219
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 224,
                      "end": 233
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 224,
                    "end": 233
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 238,
                      "end": 248
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 238,
                    "end": 248
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 253,
                      "end": 258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 253,
                    "end": 258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 263,
                      "end": 270
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 263,
                    "end": 270
                  }
                }
              ],
              "loc": {
                "start": 179,
                "end": 272
              }
            },
            "loc": {
              "start": 167,
              "end": 272
            }
          }
        ],
        "loc": {
          "start": 37,
          "end": 274
        }
      },
      "loc": {
        "start": 0,
        "end": 274
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "switchCurrentAccount",
      "loc": {
        "start": 285,
        "end": 305
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
              "start": 307,
              "end": 312
            }
          },
          "loc": {
            "start": 306,
            "end": 312
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "SwitchCurrentAccountInput",
              "loc": {
                "start": 314,
                "end": 339
              }
            },
            "loc": {
              "start": 314,
              "end": 339
            }
          },
          "loc": {
            "start": 314,
            "end": 340
          }
        },
        "directives": [],
        "loc": {
          "start": 306,
          "end": 340
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
            "value": "switchCurrentAccount",
            "loc": {
              "start": 346,
              "end": 366
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 367,
                  "end": 372
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 375,
                    "end": 380
                  }
                },
                "loc": {
                  "start": 374,
                  "end": 380
                }
              },
              "loc": {
                "start": 367,
                "end": 380
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
                  "value": "isLoggedIn",
                  "loc": {
                    "start": 388,
                    "end": 398
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 388,
                  "end": 398
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "timeZone",
                  "loc": {
                    "start": 403,
                    "end": 411
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 403,
                  "end": 411
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "users",
                  "loc": {
                    "start": 416,
                    "end": 421
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
                        "value": "activeFocusMode",
                        "loc": {
                          "start": 432,
                          "end": 447
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
                              "value": "mode",
                              "loc": {
                                "start": 462,
                                "end": 466
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
                                    "value": "filters",
                                    "loc": {
                                      "start": 485,
                                      "end": 492
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
                                            "start": 515,
                                            "end": 517
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 515,
                                          "end": 517
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "filterType",
                                          "loc": {
                                            "start": 538,
                                            "end": 548
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 538,
                                          "end": 548
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 569,
                                            "end": 572
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
                                                  "start": 599,
                                                  "end": 601
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 599,
                                                "end": 601
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 626,
                                                  "end": 636
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 626,
                                                "end": 636
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "tag",
                                                "loc": {
                                                  "start": 661,
                                                  "end": 664
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 661,
                                                "end": 664
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "bookmarks",
                                                "loc": {
                                                  "start": 689,
                                                  "end": 698
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 689,
                                                "end": 698
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 723,
                                                  "end": 735
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
                                                        "start": 766,
                                                        "end": 768
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 766,
                                                      "end": 768
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 797,
                                                        "end": 805
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 797,
                                                      "end": 805
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 834,
                                                        "end": 845
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 834,
                                                      "end": 845
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 736,
                                                  "end": 871
                                                }
                                              },
                                              "loc": {
                                                "start": 723,
                                                "end": 871
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "you",
                                                "loc": {
                                                  "start": 896,
                                                  "end": 899
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
                                                        "start": 930,
                                                        "end": 935
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 930,
                                                      "end": 935
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isBookmarked",
                                                      "loc": {
                                                        "start": 964,
                                                        "end": 976
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 964,
                                                      "end": 976
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 900,
                                                  "end": 1002
                                                }
                                              },
                                              "loc": {
                                                "start": 896,
                                                "end": 1002
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 573,
                                            "end": 1024
                                          }
                                        },
                                        "loc": {
                                          "start": 569,
                                          "end": 1024
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "focusMode",
                                          "loc": {
                                            "start": 1045,
                                            "end": 1054
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
                                                "value": "labels",
                                                "loc": {
                                                  "start": 1081,
                                                  "end": 1087
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
                                                        "start": 1118,
                                                        "end": 1120
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1118,
                                                      "end": 1120
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "color",
                                                      "loc": {
                                                        "start": 1149,
                                                        "end": 1154
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1149,
                                                      "end": 1154
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "label",
                                                      "loc": {
                                                        "start": 1183,
                                                        "end": 1188
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1183,
                                                      "end": 1188
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1088,
                                                  "end": 1214
                                                }
                                              },
                                              "loc": {
                                                "start": 1081,
                                                "end": 1214
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "schedule",
                                                "loc": {
                                                  "start": 1239,
                                                  "end": 1247
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
                                                      "value": "Schedule_common",
                                                      "loc": {
                                                        "start": 1281,
                                                        "end": 1296
                                                      }
                                                    },
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1278,
                                                      "end": 1296
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1248,
                                                  "end": 1322
                                                }
                                              },
                                              "loc": {
                                                "start": 1239,
                                                "end": 1322
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 1347,
                                                  "end": 1349
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1347,
                                                "end": 1349
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 1374,
                                                  "end": 1378
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1374,
                                                "end": 1378
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 1403,
                                                  "end": 1414
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1403,
                                                "end": 1414
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1055,
                                            "end": 1436
                                          }
                                        },
                                        "loc": {
                                          "start": 1045,
                                          "end": 1436
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 493,
                                      "end": 1454
                                    }
                                  },
                                  "loc": {
                                    "start": 485,
                                    "end": 1454
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "labels",
                                    "loc": {
                                      "start": 1471,
                                      "end": 1477
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
                                            "start": 1500,
                                            "end": 1502
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1500,
                                          "end": 1502
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "color",
                                          "loc": {
                                            "start": 1523,
                                            "end": 1528
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1523,
                                          "end": 1528
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "label",
                                          "loc": {
                                            "start": 1549,
                                            "end": 1554
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1549,
                                          "end": 1554
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 1478,
                                      "end": 1572
                                    }
                                  },
                                  "loc": {
                                    "start": 1471,
                                    "end": 1572
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminderList",
                                    "loc": {
                                      "start": 1589,
                                      "end": 1601
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
                                            "start": 1624,
                                            "end": 1626
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1624,
                                          "end": 1626
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 1647,
                                            "end": 1657
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1647,
                                          "end": 1657
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 1678,
                                            "end": 1688
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1678,
                                          "end": 1688
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminders",
                                          "loc": {
                                            "start": 1709,
                                            "end": 1718
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
                                                  "start": 1745,
                                                  "end": 1747
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1745,
                                                "end": 1747
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 1772,
                                                  "end": 1782
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1772,
                                                "end": 1782
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 1807,
                                                  "end": 1817
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1807,
                                                "end": 1817
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 1842,
                                                  "end": 1846
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1842,
                                                "end": 1846
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 1871,
                                                  "end": 1882
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1871,
                                                "end": 1882
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 1907,
                                                  "end": 1914
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1907,
                                                "end": 1914
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 1939,
                                                  "end": 1944
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1939,
                                                "end": 1944
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 1969,
                                                  "end": 1979
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1969,
                                                "end": 1979
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderItems",
                                                "loc": {
                                                  "start": 2004,
                                                  "end": 2017
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
                                                        "start": 2048,
                                                        "end": 2050
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2048,
                                                      "end": 2050
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 2079,
                                                        "end": 2089
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2079,
                                                      "end": 2089
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 2118,
                                                        "end": 2128
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2118,
                                                      "end": 2128
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 2157,
                                                        "end": 2161
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2157,
                                                      "end": 2161
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 2190,
                                                        "end": 2201
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2190,
                                                      "end": 2201
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 2230,
                                                        "end": 2237
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2230,
                                                      "end": 2237
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 2266,
                                                        "end": 2271
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2266,
                                                      "end": 2271
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 2300,
                                                        "end": 2310
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2300,
                                                      "end": 2310
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 2018,
                                                  "end": 2336
                                                }
                                              },
                                              "loc": {
                                                "start": 2004,
                                                "end": 2336
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1719,
                                            "end": 2358
                                          }
                                        },
                                        "loc": {
                                          "start": 1709,
                                          "end": 2358
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 1602,
                                      "end": 2376
                                    }
                                  },
                                  "loc": {
                                    "start": 1589,
                                    "end": 2376
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resourceList",
                                    "loc": {
                                      "start": 2393,
                                      "end": 2405
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
                                            "start": 2428,
                                            "end": 2430
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2428,
                                          "end": 2430
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 2451,
                                            "end": 2461
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2451,
                                          "end": 2461
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 2482,
                                            "end": 2494
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
                                                  "start": 2521,
                                                  "end": 2523
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2521,
                                                "end": 2523
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 2548,
                                                  "end": 2556
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2548,
                                                "end": 2556
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 2581,
                                                  "end": 2592
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2581,
                                                "end": 2592
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 2617,
                                                  "end": 2621
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2617,
                                                "end": 2621
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 2495,
                                            "end": 2643
                                          }
                                        },
                                        "loc": {
                                          "start": 2482,
                                          "end": 2643
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resources",
                                          "loc": {
                                            "start": 2664,
                                            "end": 2673
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
                                                  "start": 2700,
                                                  "end": 2702
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2700,
                                                "end": 2702
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 2727,
                                                  "end": 2732
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2727,
                                                "end": 2732
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "link",
                                                "loc": {
                                                  "start": 2757,
                                                  "end": 2761
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2757,
                                                "end": 2761
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "usedFor",
                                                "loc": {
                                                  "start": 2786,
                                                  "end": 2793
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2786,
                                                "end": 2793
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 2818,
                                                  "end": 2830
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
                                                        "start": 2861,
                                                        "end": 2863
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2861,
                                                      "end": 2863
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 2892,
                                                        "end": 2900
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2892,
                                                      "end": 2900
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 2929,
                                                        "end": 2940
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2929,
                                                      "end": 2940
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 2969,
                                                        "end": 2973
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2969,
                                                      "end": 2973
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 2831,
                                                  "end": 2999
                                                }
                                              },
                                              "loc": {
                                                "start": 2818,
                                                "end": 2999
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 2674,
                                            "end": 3021
                                          }
                                        },
                                        "loc": {
                                          "start": 2664,
                                          "end": 3021
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 2406,
                                      "end": 3039
                                    }
                                  },
                                  "loc": {
                                    "start": 2393,
                                    "end": 3039
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "schedule",
                                    "loc": {
                                      "start": 3056,
                                      "end": 3064
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
                                          "value": "Schedule_common",
                                          "loc": {
                                            "start": 3090,
                                            "end": 3105
                                          }
                                        },
                                        "directives": [],
                                        "loc": {
                                          "start": 3087,
                                          "end": 3105
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3065,
                                      "end": 3123
                                    }
                                  },
                                  "loc": {
                                    "start": 3056,
                                    "end": 3123
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 3140,
                                      "end": 3142
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3140,
                                    "end": 3142
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "name",
                                    "loc": {
                                      "start": 3159,
                                      "end": 3163
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3159,
                                    "end": 3163
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "description",
                                    "loc": {
                                      "start": 3180,
                                      "end": 3191
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3180,
                                    "end": 3191
                                  }
                                }
                              ],
                              "loc": {
                                "start": 467,
                                "end": 3205
                              }
                            },
                            "loc": {
                              "start": 462,
                              "end": 3205
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopCondition",
                              "loc": {
                                "start": 3218,
                                "end": 3231
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3218,
                              "end": 3231
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopTime",
                              "loc": {
                                "start": 3244,
                                "end": 3252
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3244,
                              "end": 3252
                            }
                          }
                        ],
                        "loc": {
                          "start": 448,
                          "end": 3262
                        }
                      },
                      "loc": {
                        "start": 432,
                        "end": 3262
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "apisCount",
                        "loc": {
                          "start": 3271,
                          "end": 3280
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 3271,
                        "end": 3280
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "bookmarkLists",
                        "loc": {
                          "start": 3289,
                          "end": 3302
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
                                "start": 3317,
                                "end": 3319
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3317,
                              "end": 3319
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "created_at",
                              "loc": {
                                "start": 3332,
                                "end": 3342
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3332,
                              "end": 3342
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "updated_at",
                              "loc": {
                                "start": 3355,
                                "end": 3365
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3355,
                              "end": 3365
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "label",
                              "loc": {
                                "start": 3378,
                                "end": 3383
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3378,
                              "end": 3383
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "bookmarksCount",
                              "loc": {
                                "start": 3396,
                                "end": 3410
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3396,
                              "end": 3410
                            }
                          }
                        ],
                        "loc": {
                          "start": 3303,
                          "end": 3420
                        }
                      },
                      "loc": {
                        "start": 3289,
                        "end": 3420
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "focusModes",
                        "loc": {
                          "start": 3429,
                          "end": 3439
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
                              "value": "filters",
                              "loc": {
                                "start": 3454,
                                "end": 3461
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
                                      "start": 3480,
                                      "end": 3482
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3480,
                                    "end": 3482
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filterType",
                                    "loc": {
                                      "start": 3499,
                                      "end": 3509
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3499,
                                    "end": 3509
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "tag",
                                    "loc": {
                                      "start": 3526,
                                      "end": 3529
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
                                            "start": 3552,
                                            "end": 3554
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3552,
                                          "end": 3554
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 3575,
                                            "end": 3585
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3575,
                                          "end": 3585
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 3606,
                                            "end": 3609
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3606,
                                          "end": 3609
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "bookmarks",
                                          "loc": {
                                            "start": 3630,
                                            "end": 3639
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3630,
                                          "end": 3639
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 3660,
                                            "end": 3672
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
                                                  "start": 3699,
                                                  "end": 3701
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3699,
                                                "end": 3701
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
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
                                                "value": "description",
                                                "loc": {
                                                  "start": 3759,
                                                  "end": 3770
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3759,
                                                "end": 3770
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3673,
                                            "end": 3792
                                          }
                                        },
                                        "loc": {
                                          "start": 3660,
                                          "end": 3792
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "you",
                                          "loc": {
                                            "start": 3813,
                                            "end": 3816
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
                                                  "start": 3843,
                                                  "end": 3848
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3843,
                                                "end": 3848
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isBookmarked",
                                                "loc": {
                                                  "start": 3873,
                                                  "end": 3885
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3873,
                                                "end": 3885
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3817,
                                            "end": 3907
                                          }
                                        },
                                        "loc": {
                                          "start": 3813,
                                          "end": 3907
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3530,
                                      "end": 3925
                                    }
                                  },
                                  "loc": {
                                    "start": 3526,
                                    "end": 3925
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "focusMode",
                                    "loc": {
                                      "start": 3942,
                                      "end": 3951
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
                                          "value": "labels",
                                          "loc": {
                                            "start": 3974,
                                            "end": 3980
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
                                                  "start": 4007,
                                                  "end": 4009
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4007,
                                                "end": 4009
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "color",
                                                "loc": {
                                                  "start": 4034,
                                                  "end": 4039
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4034,
                                                "end": 4039
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "label",
                                                "loc": {
                                                  "start": 4064,
                                                  "end": 4069
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4064,
                                                "end": 4069
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3981,
                                            "end": 4091
                                          }
                                        },
                                        "loc": {
                                          "start": 3974,
                                          "end": 4091
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "schedule",
                                          "loc": {
                                            "start": 4112,
                                            "end": 4120
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
                                                "value": "Schedule_common",
                                                "loc": {
                                                  "start": 4150,
                                                  "end": 4165
                                                }
                                              },
                                              "directives": [],
                                              "loc": {
                                                "start": 4147,
                                                "end": 4165
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4121,
                                            "end": 4187
                                          }
                                        },
                                        "loc": {
                                          "start": 4112,
                                          "end": 4187
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 4208,
                                            "end": 4210
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4208,
                                          "end": 4210
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 4231,
                                            "end": 4235
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4231,
                                          "end": 4235
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 4256,
                                            "end": 4267
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4256,
                                          "end": 4267
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3952,
                                      "end": 4285
                                    }
                                  },
                                  "loc": {
                                    "start": 3942,
                                    "end": 4285
                                  }
                                }
                              ],
                              "loc": {
                                "start": 3462,
                                "end": 4299
                              }
                            },
                            "loc": {
                              "start": 3454,
                              "end": 4299
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "labels",
                              "loc": {
                                "start": 4312,
                                "end": 4318
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
                                      "start": 4337,
                                      "end": 4339
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4337,
                                    "end": 4339
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "color",
                                    "loc": {
                                      "start": 4356,
                                      "end": 4361
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4356,
                                    "end": 4361
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "label",
                                    "loc": {
                                      "start": 4378,
                                      "end": 4383
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4378,
                                    "end": 4383
                                  }
                                }
                              ],
                              "loc": {
                                "start": 4319,
                                "end": 4397
                              }
                            },
                            "loc": {
                              "start": 4312,
                              "end": 4397
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "reminderList",
                              "loc": {
                                "start": 4410,
                                "end": 4422
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
                                      "start": 4441,
                                      "end": 4443
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4441,
                                    "end": 4443
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 4460,
                                      "end": 4470
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4460,
                                    "end": 4470
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "updated_at",
                                    "loc": {
                                      "start": 4487,
                                      "end": 4497
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4487,
                                    "end": 4497
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminders",
                                    "loc": {
                                      "start": 4514,
                                      "end": 4523
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
                                            "start": 4546,
                                            "end": 4548
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4546,
                                          "end": 4548
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 4569,
                                            "end": 4579
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4569,
                                          "end": 4579
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 4600,
                                            "end": 4610
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4600,
                                          "end": 4610
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 4631,
                                            "end": 4635
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4631,
                                          "end": 4635
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 4656,
                                            "end": 4667
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4656,
                                          "end": 4667
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "dueDate",
                                          "loc": {
                                            "start": 4688,
                                            "end": 4695
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4688,
                                          "end": 4695
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 4716,
                                            "end": 4721
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4716,
                                          "end": 4721
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isComplete",
                                          "loc": {
                                            "start": 4742,
                                            "end": 4752
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4742,
                                          "end": 4752
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderItems",
                                          "loc": {
                                            "start": 4773,
                                            "end": 4786
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
                                                  "start": 4813,
                                                  "end": 4815
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4813,
                                                "end": 4815
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 4840,
                                                  "end": 4850
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4840,
                                                "end": 4850
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 4875,
                                                  "end": 4885
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4875,
                                                "end": 4885
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 4910,
                                                  "end": 4914
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4910,
                                                "end": 4914
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 4939,
                                                  "end": 4950
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4939,
                                                "end": 4950
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 4975,
                                                  "end": 4982
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4975,
                                                "end": 4982
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 5007,
                                                  "end": 5012
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5007,
                                                "end": 5012
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 5037,
                                                  "end": 5047
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5037,
                                                "end": 5047
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4787,
                                            "end": 5069
                                          }
                                        },
                                        "loc": {
                                          "start": 4773,
                                          "end": 5069
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4524,
                                      "end": 5087
                                    }
                                  },
                                  "loc": {
                                    "start": 4514,
                                    "end": 5087
                                  }
                                }
                              ],
                              "loc": {
                                "start": 4423,
                                "end": 5101
                              }
                            },
                            "loc": {
                              "start": 4410,
                              "end": 5101
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "resourceList",
                              "loc": {
                                "start": 5114,
                                "end": 5126
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
                                      "start": 5145,
                                      "end": 5147
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5145,
                                    "end": 5147
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 5164,
                                      "end": 5174
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5164,
                                    "end": 5174
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "translations",
                                    "loc": {
                                      "start": 5191,
                                      "end": 5203
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
                                            "start": 5226,
                                            "end": 5228
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5226,
                                          "end": 5228
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "language",
                                          "loc": {
                                            "start": 5249,
                                            "end": 5257
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5249,
                                          "end": 5257
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 5278,
                                            "end": 5289
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5278,
                                          "end": 5289
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 5310,
                                            "end": 5314
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5310,
                                          "end": 5314
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5204,
                                      "end": 5332
                                    }
                                  },
                                  "loc": {
                                    "start": 5191,
                                    "end": 5332
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resources",
                                    "loc": {
                                      "start": 5349,
                                      "end": 5358
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
                                            "start": 5381,
                                            "end": 5383
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5381,
                                          "end": 5383
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 5404,
                                            "end": 5409
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5404,
                                          "end": 5409
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "link",
                                          "loc": {
                                            "start": 5430,
                                            "end": 5434
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5430,
                                          "end": 5434
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "usedFor",
                                          "loc": {
                                            "start": 5455,
                                            "end": 5462
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5455,
                                          "end": 5462
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 5483,
                                            "end": 5495
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
                                                  "start": 5522,
                                                  "end": 5524
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5522,
                                                "end": 5524
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 5549,
                                                  "end": 5557
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5549,
                                                "end": 5557
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 5582,
                                                  "end": 5593
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5582,
                                                "end": 5593
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 5618,
                                                  "end": 5622
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5618,
                                                "end": 5622
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5496,
                                            "end": 5644
                                          }
                                        },
                                        "loc": {
                                          "start": 5483,
                                          "end": 5644
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5359,
                                      "end": 5662
                                    }
                                  },
                                  "loc": {
                                    "start": 5349,
                                    "end": 5662
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5127,
                                "end": 5676
                              }
                            },
                            "loc": {
                              "start": 5114,
                              "end": 5676
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "schedule",
                              "loc": {
                                "start": 5689,
                                "end": 5697
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
                                    "value": "Schedule_common",
                                    "loc": {
                                      "start": 5719,
                                      "end": 5734
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 5716,
                                    "end": 5734
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5698,
                                "end": 5748
                              }
                            },
                            "loc": {
                              "start": 5689,
                              "end": 5748
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 5761,
                                "end": 5763
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5761,
                              "end": 5763
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "name",
                              "loc": {
                                "start": 5776,
                                "end": 5780
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5776,
                              "end": 5780
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "description",
                              "loc": {
                                "start": 5793,
                                "end": 5804
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5793,
                              "end": 5804
                            }
                          }
                        ],
                        "loc": {
                          "start": 3440,
                          "end": 5814
                        }
                      },
                      "loc": {
                        "start": 3429,
                        "end": 5814
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "handle",
                        "loc": {
                          "start": 5823,
                          "end": 5829
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5823,
                        "end": 5829
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasPremium",
                        "loc": {
                          "start": 5838,
                          "end": 5848
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5838,
                        "end": 5848
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "id",
                        "loc": {
                          "start": 5857,
                          "end": 5859
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5857,
                        "end": 5859
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "languages",
                        "loc": {
                          "start": 5868,
                          "end": 5877
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5868,
                        "end": 5877
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "membershipsCount",
                        "loc": {
                          "start": 5886,
                          "end": 5902
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5886,
                        "end": 5902
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "name",
                        "loc": {
                          "start": 5911,
                          "end": 5915
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5911,
                        "end": 5915
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "notesCount",
                        "loc": {
                          "start": 5924,
                          "end": 5934
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5924,
                        "end": 5934
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "projectsCount",
                        "loc": {
                          "start": 5943,
                          "end": 5956
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5943,
                        "end": 5956
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "questionsAskedCount",
                        "loc": {
                          "start": 5965,
                          "end": 5984
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5965,
                        "end": 5984
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "routinesCount",
                        "loc": {
                          "start": 5993,
                          "end": 6006
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5993,
                        "end": 6006
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "smartContractsCount",
                        "loc": {
                          "start": 6015,
                          "end": 6034
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 6015,
                        "end": 6034
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "standardsCount",
                        "loc": {
                          "start": 6043,
                          "end": 6057
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 6043,
                        "end": 6057
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "theme",
                        "loc": {
                          "start": 6066,
                          "end": 6071
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 6066,
                        "end": 6071
                      }
                    }
                  ],
                  "loc": {
                    "start": 422,
                    "end": 6077
                  }
                },
                "loc": {
                  "start": 416,
                  "end": 6077
                }
              }
            ],
            "loc": {
              "start": 382,
              "end": 6081
            }
          },
          "loc": {
            "start": 346,
            "end": 6081
          }
        }
      ],
      "loc": {
        "start": 342,
        "end": 6083
      }
    },
    "loc": {
      "start": 276,
      "end": 6083
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_switchCurrentAccount"
  }
} as const;
