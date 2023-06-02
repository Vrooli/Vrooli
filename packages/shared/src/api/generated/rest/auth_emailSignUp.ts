export const auth_emailSignUp = {
  "fieldName": "emailSignUp",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "emailSignUp",
        "loc": {
          "start": 328,
          "end": 339
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 340,
              "end": 345
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 348,
                "end": 353
              }
            },
            "loc": {
              "start": 347,
              "end": 353
            }
          },
          "loc": {
            "start": 340,
            "end": 353
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
                "start": 361,
                "end": 371
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 361,
              "end": 371
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 376,
                "end": 384
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 376,
              "end": 384
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 389,
                "end": 394
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
                      "start": 405,
                      "end": 420
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
                            "start": 435,
                            "end": 439
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
                                  "start": 458,
                                  "end": 465
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
                                        "start": 488,
                                        "end": 490
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 488,
                                      "end": 490
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "filterType",
                                      "loc": {
                                        "start": 511,
                                        "end": 521
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 511,
                                      "end": 521
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 542,
                                        "end": 545
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
                                              "start": 572,
                                              "end": 574
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 572,
                                            "end": 574
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 599,
                                              "end": 609
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 599,
                                            "end": 609
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tag",
                                            "loc": {
                                              "start": 634,
                                              "end": 637
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 634,
                                            "end": 637
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 662,
                                              "end": 671
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 662,
                                            "end": 671
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 696,
                                              "end": 708
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
                                                    "start": 739,
                                                    "end": 741
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 739,
                                                  "end": 741
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 770,
                                                    "end": 778
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 770,
                                                  "end": 778
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 807,
                                                    "end": 818
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 807,
                                                  "end": 818
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 709,
                                              "end": 844
                                            }
                                          },
                                          "loc": {
                                            "start": 696,
                                            "end": 844
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 869,
                                              "end": 872
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
                                                    "start": 903,
                                                    "end": 908
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 903,
                                                  "end": 908
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 937,
                                                    "end": 949
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 937,
                                                  "end": 949
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 873,
                                              "end": 975
                                            }
                                          },
                                          "loc": {
                                            "start": 869,
                                            "end": 975
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 546,
                                        "end": 997
                                      }
                                    },
                                    "loc": {
                                      "start": 542,
                                      "end": 997
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "focusMode",
                                      "loc": {
                                        "start": 1018,
                                        "end": 1027
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
                                              "start": 1054,
                                              "end": 1060
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
                                                    "start": 1091,
                                                    "end": 1093
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1091,
                                                  "end": 1093
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "color",
                                                  "loc": {
                                                    "start": 1122,
                                                    "end": 1127
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1122,
                                                  "end": 1127
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "label",
                                                  "loc": {
                                                    "start": 1156,
                                                    "end": 1161
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1156,
                                                  "end": 1161
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1061,
                                              "end": 1187
                                            }
                                          },
                                          "loc": {
                                            "start": 1054,
                                            "end": 1187
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 1212,
                                              "end": 1220
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
                                                    "start": 1254,
                                                    "end": 1269
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 1251,
                                                  "end": 1269
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1221,
                                              "end": 1295
                                            }
                                          },
                                          "loc": {
                                            "start": 1212,
                                            "end": 1295
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1320,
                                              "end": 1322
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1320,
                                            "end": 1322
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1347,
                                              "end": 1351
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1347,
                                            "end": 1351
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1376,
                                              "end": 1387
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1376,
                                            "end": 1387
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1028,
                                        "end": 1409
                                      }
                                    },
                                    "loc": {
                                      "start": 1018,
                                      "end": 1409
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 466,
                                  "end": 1427
                                }
                              },
                              "loc": {
                                "start": 458,
                                "end": 1427
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 1444,
                                  "end": 1450
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
                                        "start": 1473,
                                        "end": 1475
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1473,
                                      "end": 1475
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 1496,
                                        "end": 1501
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1496,
                                      "end": 1501
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 1522,
                                        "end": 1527
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1522,
                                      "end": 1527
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1451,
                                  "end": 1545
                                }
                              },
                              "loc": {
                                "start": 1444,
                                "end": 1545
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 1562,
                                  "end": 1574
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
                                        "start": 1597,
                                        "end": 1599
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1597,
                                      "end": 1599
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 1620,
                                        "end": 1630
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1620,
                                      "end": 1630
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 1651,
                                        "end": 1661
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1651,
                                      "end": 1661
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 1682,
                                        "end": 1691
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
                                              "start": 1718,
                                              "end": 1720
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1718,
                                            "end": 1720
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 1745,
                                              "end": 1755
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1745,
                                            "end": 1755
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 1780,
                                              "end": 1790
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1780,
                                            "end": 1790
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1815,
                                              "end": 1819
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1815,
                                            "end": 1819
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1844,
                                              "end": 1855
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1844,
                                            "end": 1855
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 1880,
                                              "end": 1887
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1880,
                                            "end": 1887
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 1912,
                                              "end": 1917
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1912,
                                            "end": 1917
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 1942,
                                              "end": 1952
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1942,
                                            "end": 1952
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 1977,
                                              "end": 1990
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
                                                    "start": 2021,
                                                    "end": 2023
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2021,
                                                  "end": 2023
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 2052,
                                                    "end": 2062
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2052,
                                                  "end": 2062
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 2091,
                                                    "end": 2101
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2091,
                                                  "end": 2101
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 2130,
                                                    "end": 2134
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2130,
                                                  "end": 2134
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 2163,
                                                    "end": 2174
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2163,
                                                  "end": 2174
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 2203,
                                                    "end": 2210
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2203,
                                                  "end": 2210
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 2239,
                                                    "end": 2244
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2239,
                                                  "end": 2244
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 2273,
                                                    "end": 2283
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2273,
                                                  "end": 2283
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1991,
                                              "end": 2309
                                            }
                                          },
                                          "loc": {
                                            "start": 1977,
                                            "end": 2309
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1692,
                                        "end": 2331
                                      }
                                    },
                                    "loc": {
                                      "start": 1682,
                                      "end": 2331
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1575,
                                  "end": 2349
                                }
                              },
                              "loc": {
                                "start": 1562,
                                "end": 2349
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 2366,
                                  "end": 2378
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
                                        "start": 2401,
                                        "end": 2403
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2401,
                                      "end": 2403
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2424,
                                        "end": 2434
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2424,
                                      "end": 2434
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 2455,
                                        "end": 2467
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
                                              "start": 2494,
                                              "end": 2496
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2494,
                                            "end": 2496
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 2521,
                                              "end": 2529
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2521,
                                            "end": 2529
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 2554,
                                              "end": 2565
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2554,
                                            "end": 2565
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 2590,
                                              "end": 2594
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2590,
                                            "end": 2594
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2468,
                                        "end": 2616
                                      }
                                    },
                                    "loc": {
                                      "start": 2455,
                                      "end": 2616
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 2637,
                                        "end": 2646
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
                                              "start": 2673,
                                              "end": 2675
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2673,
                                            "end": 2675
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 2700,
                                              "end": 2705
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2700,
                                            "end": 2705
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 2730,
                                              "end": 2734
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2730,
                                            "end": 2734
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 2759,
                                              "end": 2766
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2759,
                                            "end": 2766
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 2791,
                                              "end": 2803
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
                                                    "start": 2834,
                                                    "end": 2836
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2834,
                                                  "end": 2836
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 2865,
                                                    "end": 2873
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2865,
                                                  "end": 2873
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 2902,
                                                    "end": 2913
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2902,
                                                  "end": 2913
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 2942,
                                                    "end": 2946
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2942,
                                                  "end": 2946
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2804,
                                              "end": 2972
                                            }
                                          },
                                          "loc": {
                                            "start": 2791,
                                            "end": 2972
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2647,
                                        "end": 2994
                                      }
                                    },
                                    "loc": {
                                      "start": 2637,
                                      "end": 2994
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2379,
                                  "end": 3012
                                }
                              },
                              "loc": {
                                "start": 2366,
                                "end": 3012
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 3029,
                                  "end": 3037
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
                                        "start": 3063,
                                        "end": 3078
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 3060,
                                      "end": 3078
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3038,
                                  "end": 3096
                                }
                              },
                              "loc": {
                                "start": 3029,
                                "end": 3096
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3113,
                                  "end": 3115
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3113,
                                "end": 3115
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3132,
                                  "end": 3136
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3132,
                                "end": 3136
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 3153,
                                  "end": 3164
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3153,
                                "end": 3164
                              }
                            }
                          ],
                          "loc": {
                            "start": 440,
                            "end": 3178
                          }
                        },
                        "loc": {
                          "start": 435,
                          "end": 3178
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 3191,
                            "end": 3204
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3191,
                          "end": 3204
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 3217,
                            "end": 3225
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3217,
                          "end": 3225
                        }
                      }
                    ],
                    "loc": {
                      "start": 421,
                      "end": 3235
                    }
                  },
                  "loc": {
                    "start": 405,
                    "end": 3235
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 3244,
                      "end": 3253
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3244,
                    "end": 3253
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 3262,
                      "end": 3275
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
                            "start": 3290,
                            "end": 3292
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3290,
                          "end": 3292
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3305,
                            "end": 3315
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3305,
                          "end": 3315
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3328,
                            "end": 3338
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3328,
                          "end": 3338
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 3351,
                            "end": 3356
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3351,
                          "end": 3356
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 3369,
                            "end": 3383
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3369,
                          "end": 3383
                        }
                      }
                    ],
                    "loc": {
                      "start": 3276,
                      "end": 3393
                    }
                  },
                  "loc": {
                    "start": 3262,
                    "end": 3393
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 3402,
                      "end": 3412
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
                            "start": 3427,
                            "end": 3434
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
                                  "start": 3453,
                                  "end": 3455
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3453,
                                "end": 3455
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 3472,
                                  "end": 3482
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3472,
                                "end": 3482
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 3499,
                                  "end": 3502
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
                                        "start": 3525,
                                        "end": 3527
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3525,
                                      "end": 3527
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3548,
                                        "end": 3558
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3548,
                                      "end": 3558
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 3579,
                                        "end": 3582
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3579,
                                      "end": 3582
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 3603,
                                        "end": 3612
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3603,
                                      "end": 3612
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 3633,
                                        "end": 3645
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
                                              "start": 3672,
                                              "end": 3674
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3672,
                                            "end": 3674
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 3699,
                                              "end": 3707
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3699,
                                            "end": 3707
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3732,
                                              "end": 3743
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3732,
                                            "end": 3743
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3646,
                                        "end": 3765
                                      }
                                    },
                                    "loc": {
                                      "start": 3633,
                                      "end": 3765
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 3786,
                                        "end": 3789
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
                                              "start": 3816,
                                              "end": 3821
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3816,
                                            "end": 3821
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 3846,
                                              "end": 3858
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3846,
                                            "end": 3858
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3790,
                                        "end": 3880
                                      }
                                    },
                                    "loc": {
                                      "start": 3786,
                                      "end": 3880
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3503,
                                  "end": 3898
                                }
                              },
                              "loc": {
                                "start": 3499,
                                "end": 3898
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 3915,
                                  "end": 3924
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
                                        "start": 3947,
                                        "end": 3953
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
                                              "start": 3980,
                                              "end": 3982
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3980,
                                            "end": 3982
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 4007,
                                              "end": 4012
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4007,
                                            "end": 4012
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 4037,
                                              "end": 4042
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4037,
                                            "end": 4042
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3954,
                                        "end": 4064
                                      }
                                    },
                                    "loc": {
                                      "start": 3947,
                                      "end": 4064
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 4085,
                                        "end": 4093
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
                                              "start": 4123,
                                              "end": 4138
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 4120,
                                            "end": 4138
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4094,
                                        "end": 4160
                                      }
                                    },
                                    "loc": {
                                      "start": 4085,
                                      "end": 4160
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4181,
                                        "end": 4183
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4181,
                                      "end": 4183
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4204,
                                        "end": 4208
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4204,
                                      "end": 4208
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4229,
                                        "end": 4240
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4229,
                                      "end": 4240
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3925,
                                  "end": 4258
                                }
                              },
                              "loc": {
                                "start": 3915,
                                "end": 4258
                              }
                            }
                          ],
                          "loc": {
                            "start": 3435,
                            "end": 4272
                          }
                        },
                        "loc": {
                          "start": 3427,
                          "end": 4272
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 4285,
                            "end": 4291
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
                                  "start": 4310,
                                  "end": 4312
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4310,
                                "end": 4312
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 4329,
                                  "end": 4334
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4329,
                                "end": 4334
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 4351,
                                  "end": 4356
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4351,
                                "end": 4356
                              }
                            }
                          ],
                          "loc": {
                            "start": 4292,
                            "end": 4370
                          }
                        },
                        "loc": {
                          "start": 4285,
                          "end": 4370
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 4383,
                            "end": 4395
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
                                  "start": 4414,
                                  "end": 4416
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4414,
                                "end": 4416
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4433,
                                  "end": 4443
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4433,
                                "end": 4443
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
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
                                "value": "reminders",
                                "loc": {
                                  "start": 4487,
                                  "end": 4496
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
                                        "start": 4519,
                                        "end": 4521
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4519,
                                      "end": 4521
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4542,
                                        "end": 4552
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4542,
                                      "end": 4552
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4573,
                                        "end": 4583
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4573,
                                      "end": 4583
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4604,
                                        "end": 4608
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4604,
                                      "end": 4608
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4629,
                                        "end": 4640
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4629,
                                      "end": 4640
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 4661,
                                        "end": 4668
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4661,
                                      "end": 4668
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 4689,
                                        "end": 4694
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4689,
                                      "end": 4694
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 4715,
                                        "end": 4725
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4715,
                                      "end": 4725
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 4746,
                                        "end": 4759
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
                                              "start": 4786,
                                              "end": 4788
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4786,
                                            "end": 4788
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 4813,
                                              "end": 4823
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4813,
                                            "end": 4823
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 4848,
                                              "end": 4858
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4848,
                                            "end": 4858
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4883,
                                              "end": 4887
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4883,
                                            "end": 4887
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4912,
                                              "end": 4923
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4912,
                                            "end": 4923
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 4948,
                                              "end": 4955
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4948,
                                            "end": 4955
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 4980,
                                              "end": 4985
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4980,
                                            "end": 4985
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 5010,
                                              "end": 5020
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5010,
                                            "end": 5020
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4760,
                                        "end": 5042
                                      }
                                    },
                                    "loc": {
                                      "start": 4746,
                                      "end": 5042
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4497,
                                  "end": 5060
                                }
                              },
                              "loc": {
                                "start": 4487,
                                "end": 5060
                              }
                            }
                          ],
                          "loc": {
                            "start": 4396,
                            "end": 5074
                          }
                        },
                        "loc": {
                          "start": 4383,
                          "end": 5074
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 5087,
                            "end": 5099
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
                                  "start": 5118,
                                  "end": 5120
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5118,
                                "end": 5120
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 5137,
                                  "end": 5147
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5137,
                                "end": 5147
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 5164,
                                  "end": 5176
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
                                        "start": 5199,
                                        "end": 5201
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5199,
                                      "end": 5201
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 5222,
                                        "end": 5230
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5222,
                                      "end": 5230
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 5251,
                                        "end": 5262
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5251,
                                      "end": 5262
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 5283,
                                        "end": 5287
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5283,
                                      "end": 5287
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5177,
                                  "end": 5305
                                }
                              },
                              "loc": {
                                "start": 5164,
                                "end": 5305
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 5322,
                                  "end": 5331
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
                                        "start": 5354,
                                        "end": 5356
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5354,
                                      "end": 5356
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 5377,
                                        "end": 5382
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5377,
                                      "end": 5382
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 5403,
                                        "end": 5407
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5403,
                                      "end": 5407
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 5428,
                                        "end": 5435
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5428,
                                      "end": 5435
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5456,
                                        "end": 5468
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
                                              "start": 5495,
                                              "end": 5497
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5495,
                                            "end": 5497
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5522,
                                              "end": 5530
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5522,
                                            "end": 5530
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5555,
                                              "end": 5566
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5555,
                                            "end": 5566
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 5591,
                                              "end": 5595
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5591,
                                            "end": 5595
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5469,
                                        "end": 5617
                                      }
                                    },
                                    "loc": {
                                      "start": 5456,
                                      "end": 5617
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5332,
                                  "end": 5635
                                }
                              },
                              "loc": {
                                "start": 5322,
                                "end": 5635
                              }
                            }
                          ],
                          "loc": {
                            "start": 5100,
                            "end": 5649
                          }
                        },
                        "loc": {
                          "start": 5087,
                          "end": 5649
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 5662,
                            "end": 5670
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
                                  "start": 5692,
                                  "end": 5707
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 5689,
                                "end": 5707
                              }
                            }
                          ],
                          "loc": {
                            "start": 5671,
                            "end": 5721
                          }
                        },
                        "loc": {
                          "start": 5662,
                          "end": 5721
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5734,
                            "end": 5736
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5734,
                          "end": 5736
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5749,
                            "end": 5753
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5749,
                          "end": 5753
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5766,
                            "end": 5777
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5766,
                          "end": 5777
                        }
                      }
                    ],
                    "loc": {
                      "start": 3413,
                      "end": 5787
                    }
                  },
                  "loc": {
                    "start": 3402,
                    "end": 5787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 5796,
                      "end": 5802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5796,
                    "end": 5802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 5811,
                      "end": 5821
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5811,
                    "end": 5821
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5830,
                      "end": 5832
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5830,
                    "end": 5832
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 5841,
                      "end": 5850
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5841,
                    "end": 5850
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 5859,
                      "end": 5875
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5859,
                    "end": 5875
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5884,
                      "end": 5888
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5884,
                    "end": 5888
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 5897,
                      "end": 5907
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5897,
                    "end": 5907
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 5916,
                      "end": 5929
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5916,
                    "end": 5929
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 5938,
                      "end": 5957
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5938,
                    "end": 5957
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 5966,
                      "end": 5979
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5966,
                    "end": 5979
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractsCount",
                    "loc": {
                      "start": 5988,
                      "end": 6007
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5988,
                    "end": 6007
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 6016,
                      "end": 6030
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6016,
                    "end": 6030
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 6039,
                      "end": 6044
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6039,
                    "end": 6044
                  }
                }
              ],
              "loc": {
                "start": 395,
                "end": 6050
              }
            },
            "loc": {
              "start": 389,
              "end": 6050
            }
          }
        ],
        "loc": {
          "start": 355,
          "end": 6054
        }
      },
      "loc": {
        "start": 328,
        "end": 6054
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
      "value": "emailSignUp",
      "loc": {
        "start": 285,
        "end": 296
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
              "start": 298,
              "end": 303
            }
          },
          "loc": {
            "start": 297,
            "end": 303
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "EmailSignUpInput",
              "loc": {
                "start": 305,
                "end": 321
              }
            },
            "loc": {
              "start": 305,
              "end": 321
            }
          },
          "loc": {
            "start": 305,
            "end": 322
          }
        },
        "directives": [],
        "loc": {
          "start": 297,
          "end": 322
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
            "value": "emailSignUp",
            "loc": {
              "start": 328,
              "end": 339
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 340,
                  "end": 345
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 348,
                    "end": 353
                  }
                },
                "loc": {
                  "start": 347,
                  "end": 353
                }
              },
              "loc": {
                "start": 340,
                "end": 353
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
                    "start": 361,
                    "end": 371
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 361,
                  "end": 371
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "timeZone",
                  "loc": {
                    "start": 376,
                    "end": 384
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 376,
                  "end": 384
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "users",
                  "loc": {
                    "start": 389,
                    "end": 394
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
                          "start": 405,
                          "end": 420
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
                                "start": 435,
                                "end": 439
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
                                      "start": 458,
                                      "end": 465
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
                                            "start": 488,
                                            "end": 490
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 488,
                                          "end": 490
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "filterType",
                                          "loc": {
                                            "start": 511,
                                            "end": 521
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 511,
                                          "end": 521
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 542,
                                            "end": 545
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
                                                  "start": 572,
                                                  "end": 574
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 572,
                                                "end": 574
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 599,
                                                  "end": 609
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 599,
                                                "end": 609
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "tag",
                                                "loc": {
                                                  "start": 634,
                                                  "end": 637
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 634,
                                                "end": 637
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "bookmarks",
                                                "loc": {
                                                  "start": 662,
                                                  "end": 671
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 662,
                                                "end": 671
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 696,
                                                  "end": 708
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
                                                        "start": 739,
                                                        "end": 741
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 739,
                                                      "end": 741
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 770,
                                                        "end": 778
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 770,
                                                      "end": 778
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 807,
                                                        "end": 818
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 807,
                                                      "end": 818
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 709,
                                                  "end": 844
                                                }
                                              },
                                              "loc": {
                                                "start": 696,
                                                "end": 844
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "you",
                                                "loc": {
                                                  "start": 869,
                                                  "end": 872
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
                                                        "start": 903,
                                                        "end": 908
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 903,
                                                      "end": 908
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isBookmarked",
                                                      "loc": {
                                                        "start": 937,
                                                        "end": 949
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 937,
                                                      "end": 949
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 873,
                                                  "end": 975
                                                }
                                              },
                                              "loc": {
                                                "start": 869,
                                                "end": 975
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 546,
                                            "end": 997
                                          }
                                        },
                                        "loc": {
                                          "start": 542,
                                          "end": 997
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "focusMode",
                                          "loc": {
                                            "start": 1018,
                                            "end": 1027
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
                                                  "start": 1054,
                                                  "end": 1060
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
                                                        "start": 1091,
                                                        "end": 1093
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1091,
                                                      "end": 1093
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "color",
                                                      "loc": {
                                                        "start": 1122,
                                                        "end": 1127
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1122,
                                                      "end": 1127
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "label",
                                                      "loc": {
                                                        "start": 1156,
                                                        "end": 1161
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1156,
                                                      "end": 1161
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1061,
                                                  "end": 1187
                                                }
                                              },
                                              "loc": {
                                                "start": 1054,
                                                "end": 1187
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "schedule",
                                                "loc": {
                                                  "start": 1212,
                                                  "end": 1220
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
                                                        "start": 1254,
                                                        "end": 1269
                                                      }
                                                    },
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1251,
                                                      "end": 1269
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1221,
                                                  "end": 1295
                                                }
                                              },
                                              "loc": {
                                                "start": 1212,
                                                "end": 1295
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 1320,
                                                  "end": 1322
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1320,
                                                "end": 1322
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 1347,
                                                  "end": 1351
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1347,
                                                "end": 1351
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 1376,
                                                  "end": 1387
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1376,
                                                "end": 1387
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1028,
                                            "end": 1409
                                          }
                                        },
                                        "loc": {
                                          "start": 1018,
                                          "end": 1409
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 466,
                                      "end": 1427
                                    }
                                  },
                                  "loc": {
                                    "start": 458,
                                    "end": 1427
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "labels",
                                    "loc": {
                                      "start": 1444,
                                      "end": 1450
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
                                            "start": 1473,
                                            "end": 1475
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1473,
                                          "end": 1475
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "color",
                                          "loc": {
                                            "start": 1496,
                                            "end": 1501
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1496,
                                          "end": 1501
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "label",
                                          "loc": {
                                            "start": 1522,
                                            "end": 1527
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1522,
                                          "end": 1527
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 1451,
                                      "end": 1545
                                    }
                                  },
                                  "loc": {
                                    "start": 1444,
                                    "end": 1545
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminderList",
                                    "loc": {
                                      "start": 1562,
                                      "end": 1574
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
                                            "start": 1597,
                                            "end": 1599
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1597,
                                          "end": 1599
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 1620,
                                            "end": 1630
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1620,
                                          "end": 1630
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 1651,
                                            "end": 1661
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 1651,
                                          "end": 1661
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminders",
                                          "loc": {
                                            "start": 1682,
                                            "end": 1691
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
                                                  "start": 1718,
                                                  "end": 1720
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1718,
                                                "end": 1720
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 1745,
                                                  "end": 1755
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1745,
                                                "end": 1755
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 1780,
                                                  "end": 1790
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1780,
                                                "end": 1790
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 1815,
                                                  "end": 1819
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1815,
                                                "end": 1819
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 1844,
                                                  "end": 1855
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1844,
                                                "end": 1855
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 1880,
                                                  "end": 1887
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1880,
                                                "end": 1887
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 1912,
                                                  "end": 1917
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1912,
                                                "end": 1917
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 1942,
                                                  "end": 1952
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1942,
                                                "end": 1952
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderItems",
                                                "loc": {
                                                  "start": 1977,
                                                  "end": 1990
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
                                                        "start": 2021,
                                                        "end": 2023
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2021,
                                                      "end": 2023
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 2052,
                                                        "end": 2062
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2052,
                                                      "end": 2062
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 2091,
                                                        "end": 2101
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2091,
                                                      "end": 2101
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 2130,
                                                        "end": 2134
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2130,
                                                      "end": 2134
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 2163,
                                                        "end": 2174
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2163,
                                                      "end": 2174
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 2203,
                                                        "end": 2210
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2203,
                                                      "end": 2210
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 2239,
                                                        "end": 2244
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2239,
                                                      "end": 2244
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 2273,
                                                        "end": 2283
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2273,
                                                      "end": 2283
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1991,
                                                  "end": 2309
                                                }
                                              },
                                              "loc": {
                                                "start": 1977,
                                                "end": 2309
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1692,
                                            "end": 2331
                                          }
                                        },
                                        "loc": {
                                          "start": 1682,
                                          "end": 2331
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 1575,
                                      "end": 2349
                                    }
                                  },
                                  "loc": {
                                    "start": 1562,
                                    "end": 2349
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resourceList",
                                    "loc": {
                                      "start": 2366,
                                      "end": 2378
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
                                            "start": 2401,
                                            "end": 2403
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2401,
                                          "end": 2403
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 2424,
                                            "end": 2434
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2424,
                                          "end": 2434
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 2455,
                                            "end": 2467
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
                                                  "start": 2494,
                                                  "end": 2496
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2494,
                                                "end": 2496
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 2521,
                                                  "end": 2529
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2521,
                                                "end": 2529
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 2554,
                                                  "end": 2565
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2554,
                                                "end": 2565
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 2590,
                                                  "end": 2594
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2590,
                                                "end": 2594
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 2468,
                                            "end": 2616
                                          }
                                        },
                                        "loc": {
                                          "start": 2455,
                                          "end": 2616
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resources",
                                          "loc": {
                                            "start": 2637,
                                            "end": 2646
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
                                                  "start": 2673,
                                                  "end": 2675
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2673,
                                                "end": 2675
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 2700,
                                                  "end": 2705
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2700,
                                                "end": 2705
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "link",
                                                "loc": {
                                                  "start": 2730,
                                                  "end": 2734
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2730,
                                                "end": 2734
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "usedFor",
                                                "loc": {
                                                  "start": 2759,
                                                  "end": 2766
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2759,
                                                "end": 2766
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 2791,
                                                  "end": 2803
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
                                                        "start": 2834,
                                                        "end": 2836
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2834,
                                                      "end": 2836
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 2865,
                                                        "end": 2873
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2865,
                                                      "end": 2873
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 2902,
                                                        "end": 2913
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2902,
                                                      "end": 2913
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 2942,
                                                        "end": 2946
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2942,
                                                      "end": 2946
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 2804,
                                                  "end": 2972
                                                }
                                              },
                                              "loc": {
                                                "start": 2791,
                                                "end": 2972
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 2647,
                                            "end": 2994
                                          }
                                        },
                                        "loc": {
                                          "start": 2637,
                                          "end": 2994
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 2379,
                                      "end": 3012
                                    }
                                  },
                                  "loc": {
                                    "start": 2366,
                                    "end": 3012
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "schedule",
                                    "loc": {
                                      "start": 3029,
                                      "end": 3037
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
                                            "start": 3063,
                                            "end": 3078
                                          }
                                        },
                                        "directives": [],
                                        "loc": {
                                          "start": 3060,
                                          "end": 3078
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3038,
                                      "end": 3096
                                    }
                                  },
                                  "loc": {
                                    "start": 3029,
                                    "end": 3096
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 3113,
                                      "end": 3115
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3113,
                                    "end": 3115
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "name",
                                    "loc": {
                                      "start": 3132,
                                      "end": 3136
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3132,
                                    "end": 3136
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "description",
                                    "loc": {
                                      "start": 3153,
                                      "end": 3164
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3153,
                                    "end": 3164
                                  }
                                }
                              ],
                              "loc": {
                                "start": 440,
                                "end": 3178
                              }
                            },
                            "loc": {
                              "start": 435,
                              "end": 3178
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopCondition",
                              "loc": {
                                "start": 3191,
                                "end": 3204
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3191,
                              "end": 3204
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopTime",
                              "loc": {
                                "start": 3217,
                                "end": 3225
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3217,
                              "end": 3225
                            }
                          }
                        ],
                        "loc": {
                          "start": 421,
                          "end": 3235
                        }
                      },
                      "loc": {
                        "start": 405,
                        "end": 3235
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "apisCount",
                        "loc": {
                          "start": 3244,
                          "end": 3253
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 3244,
                        "end": 3253
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "bookmarkLists",
                        "loc": {
                          "start": 3262,
                          "end": 3275
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
                                "start": 3290,
                                "end": 3292
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3290,
                              "end": 3292
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "created_at",
                              "loc": {
                                "start": 3305,
                                "end": 3315
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3305,
                              "end": 3315
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "updated_at",
                              "loc": {
                                "start": 3328,
                                "end": 3338
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3328,
                              "end": 3338
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "label",
                              "loc": {
                                "start": 3351,
                                "end": 3356
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3351,
                              "end": 3356
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "bookmarksCount",
                              "loc": {
                                "start": 3369,
                                "end": 3383
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 3369,
                              "end": 3383
                            }
                          }
                        ],
                        "loc": {
                          "start": 3276,
                          "end": 3393
                        }
                      },
                      "loc": {
                        "start": 3262,
                        "end": 3393
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "focusModes",
                        "loc": {
                          "start": 3402,
                          "end": 3412
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
                                "start": 3427,
                                "end": 3434
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
                                      "start": 3453,
                                      "end": 3455
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3453,
                                    "end": 3455
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filterType",
                                    "loc": {
                                      "start": 3472,
                                      "end": 3482
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 3472,
                                    "end": 3482
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "tag",
                                    "loc": {
                                      "start": 3499,
                                      "end": 3502
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
                                            "start": 3525,
                                            "end": 3527
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3525,
                                          "end": 3527
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 3548,
                                            "end": 3558
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3548,
                                          "end": 3558
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 3579,
                                            "end": 3582
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3579,
                                          "end": 3582
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "bookmarks",
                                          "loc": {
                                            "start": 3603,
                                            "end": 3612
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3603,
                                          "end": 3612
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 3633,
                                            "end": 3645
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
                                                  "start": 3672,
                                                  "end": 3674
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3672,
                                                "end": 3674
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 3699,
                                                  "end": 3707
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3699,
                                                "end": 3707
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3732,
                                                  "end": 3743
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3732,
                                                "end": 3743
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3646,
                                            "end": 3765
                                          }
                                        },
                                        "loc": {
                                          "start": 3633,
                                          "end": 3765
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "you",
                                          "loc": {
                                            "start": 3786,
                                            "end": 3789
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
                                                  "start": 3816,
                                                  "end": 3821
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3816,
                                                "end": 3821
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isBookmarked",
                                                "loc": {
                                                  "start": 3846,
                                                  "end": 3858
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3846,
                                                "end": 3858
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3790,
                                            "end": 3880
                                          }
                                        },
                                        "loc": {
                                          "start": 3786,
                                          "end": 3880
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3503,
                                      "end": 3898
                                    }
                                  },
                                  "loc": {
                                    "start": 3499,
                                    "end": 3898
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "focusMode",
                                    "loc": {
                                      "start": 3915,
                                      "end": 3924
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
                                            "start": 3947,
                                            "end": 3953
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
                                                  "start": 3980,
                                                  "end": 3982
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3980,
                                                "end": 3982
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "color",
                                                "loc": {
                                                  "start": 4007,
                                                  "end": 4012
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4007,
                                                "end": 4012
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "label",
                                                "loc": {
                                                  "start": 4037,
                                                  "end": 4042
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4037,
                                                "end": 4042
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3954,
                                            "end": 4064
                                          }
                                        },
                                        "loc": {
                                          "start": 3947,
                                          "end": 4064
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "schedule",
                                          "loc": {
                                            "start": 4085,
                                            "end": 4093
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
                                                  "start": 4123,
                                                  "end": 4138
                                                }
                                              },
                                              "directives": [],
                                              "loc": {
                                                "start": 4120,
                                                "end": 4138
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4094,
                                            "end": 4160
                                          }
                                        },
                                        "loc": {
                                          "start": 4085,
                                          "end": 4160
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 4181,
                                            "end": 4183
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4181,
                                          "end": 4183
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 4204,
                                            "end": 4208
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4204,
                                          "end": 4208
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 4229,
                                            "end": 4240
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4229,
                                          "end": 4240
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3925,
                                      "end": 4258
                                    }
                                  },
                                  "loc": {
                                    "start": 3915,
                                    "end": 4258
                                  }
                                }
                              ],
                              "loc": {
                                "start": 3435,
                                "end": 4272
                              }
                            },
                            "loc": {
                              "start": 3427,
                              "end": 4272
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "labels",
                              "loc": {
                                "start": 4285,
                                "end": 4291
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
                                      "start": 4310,
                                      "end": 4312
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4310,
                                    "end": 4312
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "color",
                                    "loc": {
                                      "start": 4329,
                                      "end": 4334
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4329,
                                    "end": 4334
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "label",
                                    "loc": {
                                      "start": 4351,
                                      "end": 4356
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4351,
                                    "end": 4356
                                  }
                                }
                              ],
                              "loc": {
                                "start": 4292,
                                "end": 4370
                              }
                            },
                            "loc": {
                              "start": 4285,
                              "end": 4370
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "reminderList",
                              "loc": {
                                "start": 4383,
                                "end": 4395
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
                                      "start": 4414,
                                      "end": 4416
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4414,
                                    "end": 4416
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 4433,
                                      "end": 4443
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4433,
                                    "end": 4443
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "updated_at",
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
                                    "value": "reminders",
                                    "loc": {
                                      "start": 4487,
                                      "end": 4496
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
                                            "start": 4519,
                                            "end": 4521
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4519,
                                          "end": 4521
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 4542,
                                            "end": 4552
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4542,
                                          "end": 4552
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 4573,
                                            "end": 4583
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4573,
                                          "end": 4583
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 4604,
                                            "end": 4608
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4604,
                                          "end": 4608
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 4629,
                                            "end": 4640
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4629,
                                          "end": 4640
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "dueDate",
                                          "loc": {
                                            "start": 4661,
                                            "end": 4668
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4661,
                                          "end": 4668
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 4689,
                                            "end": 4694
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4689,
                                          "end": 4694
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isComplete",
                                          "loc": {
                                            "start": 4715,
                                            "end": 4725
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4715,
                                          "end": 4725
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderItems",
                                          "loc": {
                                            "start": 4746,
                                            "end": 4759
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
                                                  "start": 4786,
                                                  "end": 4788
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4786,
                                                "end": 4788
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 4813,
                                                  "end": 4823
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4813,
                                                "end": 4823
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 4848,
                                                  "end": 4858
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4848,
                                                "end": 4858
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 4883,
                                                  "end": 4887
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4883,
                                                "end": 4887
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 4912,
                                                  "end": 4923
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4912,
                                                "end": 4923
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 4948,
                                                  "end": 4955
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4948,
                                                "end": 4955
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 4980,
                                                  "end": 4985
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4980,
                                                "end": 4985
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 5010,
                                                  "end": 5020
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5010,
                                                "end": 5020
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4760,
                                            "end": 5042
                                          }
                                        },
                                        "loc": {
                                          "start": 4746,
                                          "end": 5042
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4497,
                                      "end": 5060
                                    }
                                  },
                                  "loc": {
                                    "start": 4487,
                                    "end": 5060
                                  }
                                }
                              ],
                              "loc": {
                                "start": 4396,
                                "end": 5074
                              }
                            },
                            "loc": {
                              "start": 4383,
                              "end": 5074
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "resourceList",
                              "loc": {
                                "start": 5087,
                                "end": 5099
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
                                      "start": 5118,
                                      "end": 5120
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5118,
                                    "end": 5120
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 5137,
                                      "end": 5147
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5137,
                                    "end": 5147
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "translations",
                                    "loc": {
                                      "start": 5164,
                                      "end": 5176
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
                                            "start": 5199,
                                            "end": 5201
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5199,
                                          "end": 5201
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "language",
                                          "loc": {
                                            "start": 5222,
                                            "end": 5230
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5222,
                                          "end": 5230
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 5251,
                                            "end": 5262
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5251,
                                          "end": 5262
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 5283,
                                            "end": 5287
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5283,
                                          "end": 5287
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5177,
                                      "end": 5305
                                    }
                                  },
                                  "loc": {
                                    "start": 5164,
                                    "end": 5305
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resources",
                                    "loc": {
                                      "start": 5322,
                                      "end": 5331
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
                                            "start": 5354,
                                            "end": 5356
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5354,
                                          "end": 5356
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 5377,
                                            "end": 5382
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5377,
                                          "end": 5382
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "link",
                                          "loc": {
                                            "start": 5403,
                                            "end": 5407
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5403,
                                          "end": 5407
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "usedFor",
                                          "loc": {
                                            "start": 5428,
                                            "end": 5435
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5428,
                                          "end": 5435
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 5456,
                                            "end": 5468
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
                                                  "start": 5495,
                                                  "end": 5497
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5495,
                                                "end": 5497
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 5522,
                                                  "end": 5530
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5522,
                                                "end": 5530
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 5555,
                                                  "end": 5566
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5555,
                                                "end": 5566
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 5591,
                                                  "end": 5595
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5591,
                                                "end": 5595
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5469,
                                            "end": 5617
                                          }
                                        },
                                        "loc": {
                                          "start": 5456,
                                          "end": 5617
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5332,
                                      "end": 5635
                                    }
                                  },
                                  "loc": {
                                    "start": 5322,
                                    "end": 5635
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5100,
                                "end": 5649
                              }
                            },
                            "loc": {
                              "start": 5087,
                              "end": 5649
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "schedule",
                              "loc": {
                                "start": 5662,
                                "end": 5670
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
                                      "start": 5692,
                                      "end": 5707
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 5689,
                                    "end": 5707
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5671,
                                "end": 5721
                              }
                            },
                            "loc": {
                              "start": 5662,
                              "end": 5721
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 5734,
                                "end": 5736
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5734,
                              "end": 5736
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "name",
                              "loc": {
                                "start": 5749,
                                "end": 5753
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5749,
                              "end": 5753
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "description",
                              "loc": {
                                "start": 5766,
                                "end": 5777
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5766,
                              "end": 5777
                            }
                          }
                        ],
                        "loc": {
                          "start": 3413,
                          "end": 5787
                        }
                      },
                      "loc": {
                        "start": 3402,
                        "end": 5787
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "handle",
                        "loc": {
                          "start": 5796,
                          "end": 5802
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5796,
                        "end": 5802
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasPremium",
                        "loc": {
                          "start": 5811,
                          "end": 5821
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5811,
                        "end": 5821
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "id",
                        "loc": {
                          "start": 5830,
                          "end": 5832
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5830,
                        "end": 5832
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "languages",
                        "loc": {
                          "start": 5841,
                          "end": 5850
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5841,
                        "end": 5850
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "membershipsCount",
                        "loc": {
                          "start": 5859,
                          "end": 5875
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5859,
                        "end": 5875
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "name",
                        "loc": {
                          "start": 5884,
                          "end": 5888
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5884,
                        "end": 5888
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "notesCount",
                        "loc": {
                          "start": 5897,
                          "end": 5907
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5897,
                        "end": 5907
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "projectsCount",
                        "loc": {
                          "start": 5916,
                          "end": 5929
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5916,
                        "end": 5929
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "questionsAskedCount",
                        "loc": {
                          "start": 5938,
                          "end": 5957
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5938,
                        "end": 5957
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "routinesCount",
                        "loc": {
                          "start": 5966,
                          "end": 5979
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5966,
                        "end": 5979
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "smartContractsCount",
                        "loc": {
                          "start": 5988,
                          "end": 6007
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5988,
                        "end": 6007
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "standardsCount",
                        "loc": {
                          "start": 6016,
                          "end": 6030
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 6016,
                        "end": 6030
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "theme",
                        "loc": {
                          "start": 6039,
                          "end": 6044
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 6039,
                        "end": 6044
                      }
                    }
                  ],
                  "loc": {
                    "start": 395,
                    "end": 6050
                  }
                },
                "loc": {
                  "start": 389,
                  "end": 6050
                }
              }
            ],
            "loc": {
              "start": 355,
              "end": 6054
            }
          },
          "loc": {
            "start": 328,
            "end": 6054
          }
        }
      ],
      "loc": {
        "start": 324,
        "end": 6056
      }
    },
    "loc": {
      "start": 276,
      "end": 6056
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_emailSignUp"
  }
};
